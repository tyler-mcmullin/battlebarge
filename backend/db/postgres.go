package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PGClient *pgxpool.Pool

// ConnectPostgres connects to the PostgreSQL database named by POSTGRES_URL,
// checks that it answers, and stores the pgx pool in PGClient
//
// Arguments: None
//
// Returns: error - non-nil if the pool cannot be created or the database does not answer a ping
func ConnectPostgres() error {
	connStr := os.Getenv("POSTGRES_URL")

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return err
	}

	PGClient = pool

	return nil
}
