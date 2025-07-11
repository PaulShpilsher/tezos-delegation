package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"tezos-delegation/internal/apperrors"
	"tezos-delegation/internal/model"
	"tezos-delegation/internal/ports"
)

// DelegationRepository implements DelegationRepositoryPort
type DelegationRepository struct {
	db *sql.DB
}

// Ensure DelegationRepository implements DelegationRepositoryPort
var _ ports.DelegationRepositoryPort = (*DelegationRepository)(nil)

func NewDelegationRepository(db *sql.DB) *DelegationRepository {
	return &DelegationRepository{db: db}
}

// InsertDelegations inserts multiple delegations into the database in a transaction.
// Returns an error if the transaction fails or if any delegation insertion fails.
func (r *DelegationRepository) InsertDelegations(delegations []*model.Delegation) (err error) {
	if len(delegations) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return apperrors.NewDatabaseErrorWithCause("begin transaction", "failed to begin transaction", err)
	}

	// Ensure transaction is rolled back on error or panic
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("panic occurred and rollback failed: %v, rollback error: %w", p, rbErr)
			} else {
				err = fmt.Errorf("panic occurred: %v", p)
			}
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("rollback failed after error: %w, rollback error: %w", err, rbErr)
			}
		}
	}()

	// Prepare statement
	const query = `INSERT INTO delegations (tzkt_id, timestamp, amount, delegator, level) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (tzkt_id) DO NOTHING`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return apperrors.NewDatabaseErrorWithCause("prepare statement", "failed to prepare insert statement", err)
	}
	defer stmt.Close()

	// Insert each delegation
	for i, d := range delegations {
		if d == nil {
			return apperrors.NewValidationError("delegation", fmt.Sprintf("delegation at index %d is nil", i))
		}

		_, err = stmt.Exec(d.TzktID, d.Timestamp, d.Amount, d.Delegator, d.Level)
		if err != nil {
			return apperrors.NewDatabaseErrorWithCause("insert delegation", fmt.Sprintf("failed to insert delegation at index %d (TzktID: %d)", i, d.TzktID), err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return apperrors.NewDatabaseErrorWithCause("commit transaction", "failed to commit transaction", err)
	}

	return nil
}

// GetLatestTzktID retrieves the highest TzktID from the database.
// Returns 0 if no delegations exist.
func (r *DelegationRepository) GetLatestTzktID(ctx context.Context) (int64, error) {
	var tzktID int64
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(tzkt_id), 0) FROM delegations").Scan(&tzktID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil // No delegations exist
		}
		return 0, apperrors.NewDatabaseErrorWithCause("query latest TzktID", "failed to get latest TzktID", err)
	}
	return tzktID, nil
}

var ErrNoDelegations = errors.New("no delegations found")

// ListDelegations retrieves delegations with pagination and optional year filtering.
// Returns ErrNoDelegations if no delegations match the criteria.
func (r *DelegationRepository) ListDelegations(ctx context.Context, limit, offset int, year *int) ([]model.Delegation, error) {
	var rows *sql.Rows
	var err error

	// Validate parameters
	if limit <= 0 {
		return nil, apperrors.NewValidationError("limit", fmt.Sprintf("must be positive, got %d", limit))
	}
	if offset < 0 {
		return nil, apperrors.NewValidationError("offset", fmt.Sprintf("must be non-negative, got %d", offset))
	}

	// Build query based on whether year filter is provided
	if year != nil {
		if *year < 2018 {
			return nil, apperrors.NewValidationError("year", fmt.Sprintf("must be a valid year from 2018 onwards, got %d", *year))
		}

		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id 
			 FROM delegations 
			 WHERE EXTRACT(YEAR FROM timestamp) = $1 
			 ORDER BY timestamp DESC, level DESC 
			 LIMIT $2 OFFSET $3`,
			*year, limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id 
			 FROM delegations 
			 ORDER BY timestamp DESC, level DESC 
			 LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}

	if err != nil {
		return nil, apperrors.NewDatabaseErrorWithCause("query delegations", "failed to query delegations", err)
	}
	defer rows.Close()

	// Scan results
	var result []model.Delegation
	for rows.Next() {
		var d model.Delegation
		if err := rows.Scan(&d.ID, &d.Timestamp, &d.Amount, &d.Delegator, &d.Level, &d.TzktID); err != nil {
			return nil, apperrors.NewDatabaseErrorWithCause("scan delegation row", "failed to scan delegation row", err)
		}
		result = append(result, d)
	}

	// Check for scan errors
	if err = rows.Err(); err != nil {
		return nil, apperrors.NewDatabaseErrorWithCause("iterate rows", "error during row iteration", err)
	}

	// Return error if no results found
	if len(result) == 0 {
		return nil, ErrNoDelegations
	}

	return result, nil
}
