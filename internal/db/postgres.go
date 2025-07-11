package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// NewDBConnectionFromDSN creates a new database connection using a DSN string
func NewDBConnectionFromDSN(dsn string) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	// Configure connection pool for better performance
	dbConn.SetMaxOpenConns(25)                 // Maximum number of open connections
	dbConn.SetMaxIdleConns(10)                 // Maximum number of idle connections
	dbConn.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection
	dbConn.SetConnMaxIdleTime(1 * time.Minute) // Maximum idle time of a connection

	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	return dbConn, nil
}
