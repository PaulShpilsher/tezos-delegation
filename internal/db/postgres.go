package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// NewDBConnectionFromDSN creates a new database connection using a DSN string
func NewDBConnectionFromDSN(dsn string) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return dbConn, nil
}
