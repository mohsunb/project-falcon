package database

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations() {
	m, err := migrate.New(
		"file://database/migrations",
		"pgx5://postgres:password@localhost:5432/postgres")
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			log.Printf("cannot perform database migration: %w\n", err)
		}
	}
}
