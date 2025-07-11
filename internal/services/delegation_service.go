package services

import (
	"context"
	"errors"
	"fmt"
	"tezos-delegation/internal/apperrors"
	"tezos-delegation/internal/db"
	"tezos-delegation/internal/model"
	"tezos-delegation/internal/ports"

	"github.com/rs/zerolog"
)

// DelegationService implements DelegationServicePort
type DelegationService struct {
	Repo   ports.DelegationRepositoryPort
	Logger zerolog.Logger
}

// Ensure DelegationService implements DelegationServicePort
var _ ports.DelegationServicePort = (*DelegationService)(nil)

func NewDelegationService(repo ports.DelegationRepositoryPort, logger zerolog.Logger) *DelegationService {
	return &DelegationService{
		Repo:   repo,
		Logger: logger.With().Str("component", "DelegationService").Logger(),
	}
}

// validatePaginationParams validates pagination parameters
func (s *DelegationService) validatePaginationParams(pageNo, pageSize int) error {
	if pageNo < 1 {
		return apperrors.NewValidationError("pageNo", fmt.Sprintf("must be positive, got %d", pageNo))
	}
	if pageSize < 1 {
		return apperrors.NewValidationError("pageSize", fmt.Sprintf("must be positive, got %d", pageSize))
	}
	if pageSize > 1000 {
		return apperrors.NewValidationError("pageSize", fmt.Sprintf("cannot exceed 1000, got %d", pageSize))
	}
	return nil
}

// validateYearParam validates the year parameter if provided
func (s *DelegationService) validateYearParam(year *int) error {
	if year != nil {
		if *year < 2018 || *year > 2100 {
			return apperrors.NewValidationError("year", fmt.Sprintf("must be between 2018 and 2100, got %d", *year))
		}
	}
	return nil
}

// GetDelegations returns delegations with pagination and optional year filter.
// Validates input parameters and handles repository errors appropriately.
func (s *DelegationService) GetDelegations(ctx context.Context, pageNo, pageSize int, year *int) ([]model.Delegation, error) {
	// Validate pagination parameters
	if err := s.validatePaginationParams(pageNo, pageSize); err != nil {
		s.Logger.Warn().Err(err).Int("pageNo", pageNo).Int("pageSize", pageSize).Msg("Invalid pagination parameters")
		return nil, fmt.Errorf("invalid pagination parameters: %w", err)
	}

	// Validate year parameter
	if err := s.validateYearParam(year); err != nil {
		s.Logger.Warn().Err(err).Interface("year", year).Msg("Invalid year parameter")
		return nil, fmt.Errorf("invalid year parameter: %w", err)
	}

	// Calculate offset
	offset := (pageNo - 1) * pageSize

	// Get delegations from repository
	delegations, err := s.Repo.ListDelegations(ctx, pageSize, offset, year)
	if err != nil {
		// Handle specific repository errors
		if errors.Is(err, db.ErrNoDelegations) {
			s.Logger.Info().Int("pageNo", pageNo).Int("pageSize", pageSize).Interface("year", year).Msg("No delegations found")
			return []model.Delegation{}, nil
		}

		s.Logger.Error().Err(err).Int("pageNo", pageNo).Int("pageSize", pageSize).Interface("year", year).Msg("Repository error in GetDelegations")
		return nil, fmt.Errorf("failed to retrieve delegations: %w", err)
	}

	s.Logger.Debug().Int("count", len(delegations)).Int("pageNo", pageNo).Int("pageSize", pageSize).Interface("year", year).Msg("Retrieved delegations")
	return delegations, nil
}
