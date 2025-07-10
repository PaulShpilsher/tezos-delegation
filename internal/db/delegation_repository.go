package db

import (
	"context"
	"database/sql"
	"errors"
	"tezos-delegation/internal/model"
)

// DelegationRepositoryInterface defines the contract for delegation data persistence
// (You can remove the go:generate line if you don't use mockgen)
//
//go:generate mockgen -destination=../mocks/mock_delegation_repository.go -package=mocks tezos-delegation/internal/db DelegationRepositoryInterface
type DelegationRepositoryInterface interface {
	InsertDelegations(delegations []*model.Delegation) error
	GetLatestTzktID(ctx context.Context) (int64, error)
	ListDelegations(ctx context.Context, limit, offset int, year *int) ([]model.Delegation, error)
}

type DelegationRepository struct {
	db *sql.DB
}

func NewDelegationRepository(db *sql.DB) *DelegationRepository {
	return &DelegationRepository{db: db}
}

func (r *DelegationRepository) InsertDelegations(delegations []*model.Delegation) (err error) {
	if len(delegations) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	const query = `INSERT INTO delegations (tzkt_id, timestamp, amount, delegator, level) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (tzkt_id) DO NOTHING`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range delegations {
		_, err = stmt.Exec(d.TzktID, d.Timestamp, d.Amount, d.Delegator, d.Level)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}

func (r *DelegationRepository) GetLatestTzktID(ctx context.Context) (int64, error) {

	var tzktID int64
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(tzkt_id), 0) FROM delegations").Scan(&tzktID)
	if err != nil {
		return 0, err
	}
	return tzktID, nil
}

var ErrNoDelegations = errors.New("no delegations found")

func (r *DelegationRepository) ListDelegations(ctx context.Context, limit, offset int, year *int) ([]model.Delegation, error) {
	var rows *sql.Rows
	var err error
	if year != nil {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id FROM delegations WHERE EXTRACT(YEAR FROM timestamp) = $1 ORDER BY timestamp DESC, level DESC LIMIT $2 OFFSET $3`,
			*year, limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id FROM delegations ORDER BY timestamp DESC, level DESC LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Delegation
	for rows.Next() {
		var d model.Delegation
		if err := rows.Scan(&d.ID, &d.Timestamp, &d.Amount, &d.Delegator, &d.Level, &d.TzktID); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if len(result) == 0 {
		return nil, ErrNoDelegations
	}
	return result, nil
}
