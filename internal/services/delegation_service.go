package services

import (
	"context"
	"time"

	"tezoz-delegation/internal/db"
	"tezoz-delegation/internal/model"
)

type DelegationService struct {
	Repo *db.DelegationRepository
}

func NewDelegationService(repo *db.DelegationRepository) *DelegationService {
	return &DelegationService{Repo: repo}
}

// GetDelegations returns delegations with pagination and optional year filter.
func (s *DelegationService) GetDelegations(ctx context.Context, page int, year string) ([]model.Delegation, error) {
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	var from, to time.Time
	var err error
	if year != "" {
		from, err = time.Parse("2006", year)
		if err != nil {
			return nil, err
		}
		to = from.AddDate(1, 0, 0)
	}

	return s.Repo.ListDelegations(ctx, limit, offset, from, to)
}
