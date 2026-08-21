package db

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/jmoiron/sqlx"
)

// driverName is the database/sql driver used to connect to PostgreSQL. It is
// also handed to sqlx so that sqlx keeps using the pgx bindvar style
// ($1, $2, ...).
const driverName = "pgx"

// Open connects to PostgreSQL through the pgx driver and verifies the
// connection.
//
// It is the single entry point for opening the database, shared by the server
// and the repository tests, so that driver registration and connection checks
// live next to the repositories that depend on them.
func Open(ctx context.Context, databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Open(driverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping database: %w; close database: %v", err, closeErr)
		}

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
