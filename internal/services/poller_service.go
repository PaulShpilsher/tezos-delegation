package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"sync"
	"tezoz-delegation/internal/db"
	"tezoz-delegation/internal/model"
)

const (
	tzktBaseURL     = "https://api.tzkt.io/v1/operations/delegations"
	pageSize        = 1000        // Tzkt max page size
	rateLimit       = time.Second // 1 request per second
	maxRetries      = 5
	initialBackoff  = time.Second
	maxErrorBodyLen = 4096
	maxTotalWait    = 2 * time.Minute
)

type Poller struct {
	repo   *db.DelegationRepository
	client *http.Client
	wg     sync.WaitGroup
}

func NewPoller(repo *db.DelegationRepository) *Poller {
	return &Poller{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type tzktDelegation struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Amount    int64     `json:"amount"`
	Sender    struct {
		Address string `json:"address"`
	} `json:"sender"`
	Level int64 `json:"level"`
}

func (p *Poller) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.syncAndPoll(ctx)
}

// Wait blocks until the poller has fully stopped.
func (p *Poller) Wait() {
	p.wg.Wait()
}

// syncAndPoll first downloads all historical data as fast as possible (rate-limited),
// then switches to periodic polling for new data every minute, catching up if behind.
func (p *Poller) syncAndPoll(ctx context.Context) {
	defer p.wg.Done()
	// 1. Historical sync: fast as possible within rate limits
	log.Println("[poller] sync historical data")
	for {
		caughtUp, err := p.syncDelegationsBatch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[poller] context cancelled during historical sync, exiting")
				return
			}
			log.Printf("[poller] error during historical sync: %v", err)
			time.Sleep(time.Millisecond * 500)
			continue
		}
		if caughtUp {
			break
		}
	}

	// 2. Polling: every minute, but catch up if behind
	log.Println("[poller] synced.  polling for new data")
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				caughtUp, err := p.syncDelegationsBatch(ctx)
				if err != nil {
					if ctx.Err() != nil {
						log.Println("[poller] context cancelled during polling, exiting")
						return
					}
					log.Printf("[poller] error during polling: %v", err)
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

func (p *Poller) syncDelegationsBatch(ctx context.Context) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// Get last stored delegaton TztkID
	lastTzktID, err := p.repo.GetLatestTzktID(ctx)
	if err != nil {
		return false, fmt.Errorf("error getting latest TzktID from database: %w", err)
	}

	log.Printf("[poller] fetching delegations with latest TzktID %d", lastTzktID)

	// Get a block of delegations from Tztk API
	delegations, err := p.fetchDelegationBatch(ctx, lastTzktID)
	if err != nil {
		return false, fmt.Errorf("error fetching delegations from tzkt: %w", err)

	}

	log.Printf("[poller] fetched %d delegations", len(delegations))
	if len(delegations) == 0 {
		return true, nil // caught up
	}

	// Convert []model.Delegation to []*model.Delegation
	delegationPtrs := make([]*model.Delegation, len(delegations))
	for i := range delegations {
		delegationPtrs[i] = &delegations[i]
	}

	err = p.repo.InsertDelegations(delegationPtrs)
	if err != nil {
		return false, fmt.Errorf("error storing delegations ro database: %w", err)
	}
	return len(delegations) < pageSize, nil // caught up if less than a full page
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

// fetchDelegationBatch fetches a batch of delegations from the Tzkt API, handling rate limits, server errors, and retries.
//
// - Retries on HTTP 429 (Too Many Requests) and 503 (Service Unavailable), respecting the Retry-After header if present.
// - Retries on all 5xx server errors with exponential backoff.
// - Fails fast on other non-200 status codes, logging the response body for diagnostics.
// - Enforces a maximum number of retries and a maximum total wait time.
// - All network and retry waits are cancellable via the provided context.
func (p *Poller) fetchDelegationBatch(ctx context.Context, lastID int64) ([]model.Delegation, error) {

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
				log.Printf("[poller] HTTP %d, Retry-After: %s, waiting %s (attempt %d/%d)", resp.StatusCode, retryAfter, wait, attempt+1, maxRetries)
			} else {
				// Fallback to exponential backoff if Retry-After is missing or invalid
				wait = backoff
				log.Printf("[poller] HTTP %d, invalid/missing Retry-After, backoff %s (attempt %d/%d)", resp.StatusCode, wait, attempt+1, maxRetries)
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
				log.Printf("[poller] HTTP %d server error, retrying in %s (attempt %d/%d)", resp.StatusCode, backoff, attempt+1, maxRetries)
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
			log.Printf("[poller] HTTP %d unexpected, not retrying. Body: %s", resp.StatusCode, string(body))
			return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}
	}
	if resp == nil {
		return nil, fmt.Errorf("nil resoponse: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON response into tzktDelegation structs
	var result []tzktDelegation
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error decoding response body: %w", err)
	}

	// Convert to model.Delegation slice
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
