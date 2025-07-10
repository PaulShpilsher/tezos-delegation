package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"tezoz-delegation/internal/db"
	"tezoz-delegation/internal/model"
)

const (
	tzktBaseURL = "https://api.tzkt.io/v1/operations/delegations"
	pageSize    = 1000        // Tzkt max page size
	rateLimit   = time.Second // 1 request per second
	maxRetries  = 5
)

type Poller struct {
	repo   *db.DelegationRepository
	client *http.Client
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
	go p.syncAndPoll(ctx)
}

// syncAndPoll first downloads all historical data as fast as possible (rate-limited),
// then switches to periodic polling for new data every minute, catching up if behind.
func (p *Poller) syncAndPoll(ctx context.Context) {
	// 1. Historical sync: fast as possible within rate limits
	for {
		caughtUp, err := p.syncDelegationsBatch(ctx)
		if err != nil {
			log.Printf("[poller] error during historical sync: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if caughtUp {
			break
		}
	}

	// 2. Polling: every minute, but catch up if behind
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

	// Get last stored delegaton TztkID
	lastTzktID, err := p.repo.GetLatestTzktID()
	if err != nil {
		return false, err
	}

	// Get a block of delegations from Tztk API
	delegations, err := p.fetchDelegationBatch(lastTzktID)
	if err != nil {
		return false, err
	}
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
		return false, err
	}
	return len(delegations) < pageSize, nil // caught up if less than a full page
}

func (p *Poller) fetchDelegationBatch(lastID int64) ([]model.Delegation, error) {

	url := fmt.Sprintf("%s?limit=%d&id.gt=%d", tzktBaseURL, pageSize, lastID)

	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	var result []tzktDelegation
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

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
