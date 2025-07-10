package db

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"tezos-delegation/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	cleanup := func() { db.Close() }
	return db, mock, cleanup
}

func TestInsertDelegations(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()
	repo := NewDelegationRepository(db)
	delegations := []*model.Delegation{{TzktID: 1, Timestamp: fixedTime(), Amount: 100, Delegator: "tz1", Level: 1}}

	mock.ExpectBegin()
	mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO delegations (tzkt_id, timestamp, amount, delegator, level) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (tzkt_id) DO NOTHING`)).
		ExpectExec().
		WithArgs(delegations[0].TzktID, delegations[0].Timestamp, delegations[0].Amount, delegations[0].Delegator, delegations[0].Level).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.InsertDelegations(delegations)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLatestTzktID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()
	repo := NewDelegationRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(MAX(tzkt_id), 0) FROM delegations")).
		WillReturnRows(sqlmock.NewRows([]string{"tzkt_id"}).AddRow(42))

	id, err := repo.GetLatestTzktID(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListDelegations(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()
	repo := NewDelegationRepository(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "timestamp", "amount", "delegator", "level", "tzkt_id"}).
		AddRow(1, fixedTime(), 100, "tz1", 1, 1)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, timestamp, amount, delegator, level, tzkt_id FROM delegations ORDER BY timestamp DESC, level DESC LIMIT $1 OFFSET $2`)).
		WithArgs(10, 0).
		WillReturnRows(rows)

	delegations, err := repo.ListDelegations(ctx, 10, 0, nil)
	assert.NoError(t, err)
	assert.Len(t, delegations, 1)
	assert.Equal(t, "tz1", delegations[0].Delegator)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// fixedTime returns a constant time.Time for use in tests
func fixedTime() time.Time {
	return time.Date(2022, 5, 5, 6, 29, 14, 0, time.UTC)
}
