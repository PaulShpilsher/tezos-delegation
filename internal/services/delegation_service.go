package services

import (
	"context"

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
func (s *DelegationService) GetDelegations(ctx context.Context, pageNo, pageSize int, year *int) ([]model.Delegation, error) {
	if pageNo < 1 {
		pageNo = 1
	}
	offset := (pageNo - 1) * pageSize

	delegations, err := s.Repo.ListDelegations(ctx, pageSize, offset, year)
	if err != nil {
		if err == db.ErrNoDelegations {
			return []model.Delegation{}, nil
		}
		return nil, err
	}
	return delegations, nil
}
