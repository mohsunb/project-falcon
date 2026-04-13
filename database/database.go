package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func PrepareDatabase(ctx context.Context, logger *slog.Logger, username string, password string, host string, port int, database string) (*pgxpool.Pool, error) {
	err := runMigrations(username, password, host, port, database)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("cannot perform database migration: %w", err)
	}

	pool, err := pgxpool.New(ctx, fmt.Sprintf("postgres://%s:%s@%s:%d/%s", username, password, host, port, database))
	if err != nil {
		return nil, fmt.Errorf("failed to establish db connection: %w", err)
	}
	return pool, nil
}

func runMigrations(username string, password string, host string, port int, database string) error {
	m, err := migrate.New("file://database/migrations", fmt.Sprintf("pgx5://%s:%s@%s:%d/%s", username, password, host, port, database))
	if err != nil {
		return err
	}
	return m.Up()
}
