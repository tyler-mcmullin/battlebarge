package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PGClient *pgxpool.Pool

// No arguments, no returns. Function connects to a PostgreSQL database
// and creates a pgx Pool struct to interact with the database
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
