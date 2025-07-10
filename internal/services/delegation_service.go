package services

import (
	"context"

	"tezos-delegation/internal/db"
	"tezos-delegation/internal/model"

	"github.com/rs/zerolog"
)

type DelegationService struct {
	Repo   *db.DelegationRepository
	Logger zerolog.Logger
}

func NewDelegationService(repo *db.DelegationRepository, logger zerolog.Logger) *DelegationService {
	return &DelegationService{Repo: repo, Logger: logger}
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
			s.Logger.Warn().Msg("No delegations found for the given parameters")
			return []model.Delegation{}, nil
		}
		return nil, err
	}
	return delegations, nil
}
