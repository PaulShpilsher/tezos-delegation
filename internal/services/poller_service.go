package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"sync"
	"tezos-delegation/internal/model"
	"tezos-delegation/internal/ports"

	"github.com/rs/zerolog"
)

const (
	tzktBaseURL     = "https://api.tzkt.io/v1/operations/delegations"
	pageSize        = 1000 // Tzkt max page size
	maxRetries      = 5
	initialBackoff  = time.Second
	maxErrorBodyLen = 4096
	maxTotalWait    = 2 * time.Minute
)

// PollerService periodically syncs delegation data from the Tzkt API to the local database.
type PollerService struct {
	repo   ports.DelegationRepositoryPort // Use interface for easier mocking
	client *http.Client                   // HTTP client for making API requests
	wg     sync.WaitGroup                 // WaitGroup to manage goroutine lifecycle
	logger zerolog.Logger                 // Structured logger for logging events and errors
}

// NewPoller constructs a new Poller instance with the provided repository and logger.
func NewPoller(repo ports.DelegationRepositoryPort, logger zerolog.Logger) *PollerService {
	// Configure HTTP client with connection pooling and timeouts
	transport := &http.Transport{
		MaxIdleConns:        100,              // Maximum idle connections
		MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // How long to keep idle connections
		TLSHandshakeTimeout: 10 * time.Second, // TLS handshake timeout
		DisableCompression:  false,            // Enable compression
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // Set a reasonable timeout for API requests
	}

	return &PollerService{
		repo:   repo,
		client: client,
		logger: logger.With().Str("component", "PollerService").Logger(),
	}
}

// tzktDelegation represents the structure of a delegation operation returned by the Tzkt API.
type tzktDelegation struct {
	ID        int64     `json:"id"`        // Unique operation ID in Tzkt
	Timestamp time.Time `json:"timestamp"` // Time of the delegation operation
	Amount    int64     `json:"amount"`    // Amount delegated (mutez)
	Sender    struct {
		Address string `json:"address"` // Delegator's address
	} `json:"sender"`
	Level int64 `json:"level"` // Block level of the operation
}

// Start launches the poller in a new goroutine, beginning the sync and poll process.
// The context is used for cancellation and shutdown.
func (p *PollerService) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.syncAndPoll(ctx)
}

// Wait blocks until the poller has fully stopped (i.e., the goroutine has exited).
func (p *PollerService) Wait() {
	p.wg.Wait()
}

// syncAndPoll first downloads all historical data as fast as possible (rate-limited),
// then switches to periodic polling for new data every minute, catching up if behind.
func (p *PollerService) syncAndPoll(ctx context.Context) {
	defer p.wg.Done()
	// 1. Historical sync: fast as possible within rate limits
	p.logger.Info().Str("phase", "historical_sync").Msg("syncing historical data")
	for {
		// Attempt to fetch and store a batch of delegations
		caughtUp, err := p.syncDelegationsBatch(ctx)
		if err != nil {
			// If the context was cancelled, log and exit immediately
			if ctx.Err() != nil {
				p.logger.Error().Err(err).Str("phase", "historical_sync").Msg("context cancelled during historical sync, exiting")
				return
			}
			// Log the error and retry after a short delay
			p.logger.Error().Err(err).Str("phase", "historical_sync").Msg("error during historical sync")
			time.Sleep(time.Second)
			continue
		}
		// If caught up (no more historical data), break out of the loop
		if caughtUp {
			break
		}
	}

	// 2. Polling: every minute, but catch up if behind
	p.logger.Info().Msg("caught up. Polling for new data...")
	ticker := time.NewTicker(time.Minute) // Poll every minute
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Context cancelled: exit polling loop
			return
		case <-ticker.C:
			// On each tick, try to catch up (in case multiple batches are needed)
			for {
				caughtUp, err := p.syncDelegationsBatch(ctx)
				if err != nil {
					if ctx.Err() != nil {
						p.logger.Error().Err(err).Str("phase", "polling").Msg("context cancelled during polling, exiting")
						return
					}
					p.logger.Error().Err(err).Str("phase", "polling").Msg("error during polling")
					time.Sleep(time.Second)
					continue
				}
				if caughtUp {
					break
				}
			}
		}
	}
}

// syncDelegationsBatch fetches a batch of new delegations from Tzkt and stores them in the database.
// Returns (caughtUp, error): caughtUp is true if there are no more new delegations to fetch.
func (p *PollerService) syncDelegationsBatch(ctx context.Context) (bool, error) {
	if ctx.Err() != nil {
		return false, fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// Get last stored delegation TzktID from the database to avoid duplicates
	lastTzktID, err := p.repo.GetLatestTzktID(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get latest TzktID from database: %w", err)
	}

	// Fetch a batch of delegations from the Tzkt API, starting after lastTzktID
	delegations, err := p.fetchDelegationBatch(ctx, lastTzktID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch delegations from Tzkt API: %w", err)
	}

	p.logger.Info().Int("fetched_delegations_count", len(delegations)).Int64("last_tzkt_id", lastTzktID).Msg("Fetched delegation batch")
	if len(delegations) == 0 {
		return true, nil // caught up: no new delegations
	}

	// Convert []model.Delegation to []*model.Delegation for database insertion
	delegationPtrs := make([]*model.Delegation, len(delegations))
	for i := range delegations {
		delegationPtrs[i] = &delegations[i]
	}

	// Insert the new delegations into the database
	err = p.repo.InsertDelegations(delegationPtrs)
	if err != nil {
		return false, fmt.Errorf("failed to store delegations to database: %w", err)
	}

	// If less than a full page was fetched, we're caught up; otherwise, there may be more
	return len(delegations) < pageSize, nil
}

// fetchDelegationBatch fetches a batch of delegations from the Tzkt API, handling rate limits, server errors, and retries.
//
// - Retries on HTTP 429 (Too Many Requests) and 503 (Service Unavailable), respecting the Retry-After header if present.
// - Retries on all 5xx server errors with exponential backoff.
// - Fails fast on other non-200 status codes, logging the response body for diagnostics.
// - Enforces a maximum number of retries and a maximum total wait time.
// - All network and retry waits are cancellable via the provided context.
func (p *PollerService) fetchDelegationBatch(ctx context.Context, lastID int64) ([]model.Delegation, error) {
	// Construct the Tzkt API URL with pagination (id.gt=lastID)
	url := fmt.Sprintf("%s?limit=%d&id.gt=%d", tzktBaseURL, pageSize, lastID)

	var resp *http.Response
	var err error
	backoff := initialBackoff
	start := time.Now()

retryLoop:
	for attempt := 0; attempt < maxRetries && time.Since(start) < maxTotalWait; attempt++ {
		// Create a new HTTP request with context for cancellation/timeout support
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if reqErr != nil {
			return nil, reqErr
		}
		resp, err = p.client.Do(req)
		if err != nil {
			// Network error or context cancellation
			return nil, err
		}

		// Handle HTTP status codes
		switch resp.StatusCode {
		case http.StatusOK:
			// Success: break out of retry loop and process response
			break retryLoop
		case http.StatusTooManyRequests, http.StatusServiceUnavailable:
			// Rate limited or temporarily unavailable: check Retry-After header
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()
			var wait time.Duration
			if d, err := parseRetryAfter(retryAfter); err == nil && d > 0 {
				wait = d
				p.logger.Info().Int("status_code", resp.StatusCode).Str("retry_after", retryAfter).Dur("wait_time", wait).Msg("HTTP status too many requests, retrying in")
			} else {
				// Fallback to exponential backoff if Retry-After is missing or invalid
				wait = backoff
				p.logger.Info().Int("status_code", resp.StatusCode).Dur("wait_time", wait).Int("attempt", attempt+1).Int("max_retries", maxRetries).Msg("HTTP status too many requests, invalid/missing Retry-After, backoff")
				backoff *= 2
			}
			// Wait for the specified duration or until context is cancelled
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			continue
		default:
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				// Server error: retry with exponential backoff
				resp.Body.Close()
				p.logger.Info().Int("status_code", resp.StatusCode).Int("attempt", attempt+1).Int("max_retries", maxRetries).Dur("wait_time", backoff).Msg("HTTP server error, retrying in")
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
				backoff *= 2
				continue
			}
			// Other unexpected status codes: log and return error with response body
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyLen))
			resp.Body.Close()
			p.logger.Error().Int("status_code", resp.StatusCode).Str("body", string(body)).Msg("HTTP unexpected, not retrying")
			return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON response into tzktDelegation structs
	var result []tzktDelegation
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	// Convert to model.Delegation slice for database storage
	delegations := make([]model.Delegation, len(result))
	for i, op := range result {
		delegations[i] = model.Delegation{
			TzktID:    op.ID,
			Timestamp: op.Timestamp,
			Amount:    op.Amount,
			Delegator: op.Sender.Address,
			Level:     op.Level,
		}
	}

	return delegations, nil
}

// parseRetryAfter parses the Retry-After header, supporting both seconds and HTTP-date formats.
// Returns a duration to wait, or an error if the header is missing or invalid.
func parseRetryAfter(header string) (time.Duration, error) {
	if header == "" {
		return 0, fmt.Errorf("empty Retry-After")
	}
	// Try as integer seconds
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second, nil
	}
	// Try as HTTP-date
	if t, err := http.ParseTime(header); err == nil {
		return time.Until(t), nil
	}
	return 0, fmt.Errorf("invalid Retry-After: %s", header)
}
