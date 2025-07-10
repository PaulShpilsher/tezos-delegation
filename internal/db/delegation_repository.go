package db

import (
	"context"
	"database/sql"
	"tezoz-delegation/internal/model"
	"time"
)

type DelegationRepository struct {
	db *sql.DB
}

func NewDelegationRepository(db *sql.DB) *DelegationRepository {
	return &DelegationRepository{db: db}
}

func (r *DelegationRepository) InsertDelegation(d *model.Delegation) error {
	_, err := r.db.Exec(`
		INSERT INTO delegations (tzkt_id, timestamp, amount, delegator, level)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tzkt_id) DO NOTHING
	`, d.TzktID, d.Timestamp, d.Amount, d.Delegator, d.Level)

	return err
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

func (r *DelegationRepository) ListDelegations(ctx context.Context, limit, offset int, from, to time.Time) ([]model.Delegation, error) {
	var rows *sql.Rows
	var err error
	if from.IsZero() {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id FROM delegations ORDER BY timestamp DESC, level DESC LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(
			ctx,
			`SELECT id, timestamp, amount, delegator, level, tzkt_id FROM delegations WHERE timestamp >= $1 AND timestamp < $2 ORDER BY timestamp DESC, level DESC LIMIT $3 OFFSET $4`,
			from, to, limit, offset,
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
	return result, nil
}
