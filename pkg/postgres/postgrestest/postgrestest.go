package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/agalitsyn/goth/pkg/postgres"
	"github.com/agalitsyn/goth/pkg/secret"
)

func SetupTestDB(t *testing.T) (*postgres.DB, func()) {
	return setupTestDBWithName(t, t.Name())
}

func setupTestDBWithName(t *testing.T, name string) (*postgres.DB, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip long-running test in short mode")
	}

	dsn, ok := os.LookupEnv("DATABASE_URI")
	if !ok {
		panic("database uri not defined")
	}
	cfg := postgres.Config{URI: secret.NewString(dsn)}

	testDB := dbName(name)

	connCfg, err := pgx.ParseConfig(cfg.URI.Unmask())
	require.NoError(t, err)
	connCfg.Database = testDB

	dr, err := sql.Open(
		postgres.DriverName,
		stdlib.RegisterConnConfig(connCfg),
	)
	require.NoError(t, err)

	dbx := sqlx.NewDb(dr, postgres.DriverName)
	db := &postgres.DB{
		Session: dbx,
		Cfg:     cfg,
	}

	ctx := context.Background()
	managementConnCfg := connCfg.Copy()
	err = dbCreate(ctx, managementConnCfg, testDB)
	require.NoError(t, err)

	err = db.PingContext(ctx)
	require.NoError(t, err)

	teardown := func() {
		err = db.Session.Close()
		require.NoError(t, err)

		err = dbDrop(ctx, managementConnCfg, testDB)
		require.NoError(t, err)
	}

	return db, teardown
}

var testDBIdx uint32

func dbName(name string) string {
	name = strings.ToLower(strings.Replace(name, "/", "_", -1))
	idx := atomic.AddUint32(&testDBIdx, 1)
	name = strings.ToLower(fmt.Sprintf("test_%d_%s", idx, name))
	return fmt.Sprintf("test_%d_%s", time.Now().Unix(), name)
}

const managementDB = "postgres"

func dbCreate(ctx context.Context, cfg *pgx.ConnConfig, name string) error {
	cfg.Database = managementDB
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	if _, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", name)); err != nil {
		return err
	}
	if err = conn.Close(ctx); err != nil {
		return err
	}
	return nil
}

func dbDrop(ctx context.Context, cfg *pgx.ConnConfig, name string) error {
	cfg.Database = managementDB
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", name))
	if err != nil {
		return err
	}
	if err = conn.Close(ctx); err != nil {
		return err
	}
	return nil
}

func CountRows(t *testing.T, table string, db *postgres.DB) int {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	err := db.Session.QueryRow(query).Scan(&count)
	require.NoError(t, err)
	return count
}

// NowLocalTime returns time with fixed zone
// Using this helper in test fixtures allows to compare structs after fetching from storage
// Avoids lib/pq issue with `+0000` vs stdlib.Time `UTC`
func NowLocalTime() time.Time {
	return time.Now().Round(time.Microsecond).Local()
}
