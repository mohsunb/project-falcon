package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func PrepareDatabase(ctx context.Context, username string, password string, host string, port int, database string) (*pgxpool.Pool, error) {
	RunMigrations()
	pool, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", username, password, host, port, database))
	if err != nil {
		return nil, fmt.Errorf("failed to establish db connection: %w", err)
	} else {
		log.Println("successfully acquired db connection")
	}
	return pool, nil
}
