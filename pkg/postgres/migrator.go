package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
)

const migrationsTableName = "schema_version"

func logMigrationProcess(version int32, name, direction, _ string) {
	slog.Info("migrating", "version", version, "name", name, "direction", direction)
}

func MigrateUp(ctx context.Context, conn *pgx.Conn, migrationsFiles fs.FS) error {
	m, err := migrate.NewMigrator(ctx, conn, migrationsTableName)
	if err != nil {
		return fmt.Errorf("could not create migrator: %w", err)
	}

	if err = m.LoadMigrations(migrationsFiles); err != nil {
		return fmt.Errorf("could not load migrations: %w", err)
	}

	m.OnStart = logMigrationProcess
	if err = m.Migrate(ctx); err != nil {
		return fmt.Errorf("could not migrate: %w", err)
	}

	return nil
}
