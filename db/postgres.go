package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var PGClient *pgxpool.Pool

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
