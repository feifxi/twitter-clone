package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrations are embedded so the binary doesn't depend on its working directory.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies all pending up migrations.
func Migrate(databaseURL string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
