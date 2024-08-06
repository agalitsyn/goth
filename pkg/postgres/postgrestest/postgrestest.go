package postgrestest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/require"

	"github.com/agalitsyn/goth/migrations"
	"github.com/agalitsyn/goth/pkg/postgres"
)

func SetupTestDB(t *testing.T) (postgres.Querier, func()) {
	return setupTestDBWithName(t, t.Name())
}

func setupTestDBWithName(t *testing.T, name string) (postgres.Querier, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip postgres test in short mode")
	}

	dsn, ok := os.LookupEnv("TEST_POSTGRES_URI")
	if !ok {
		panic("database uri not defined")
	}
	cfg := postgres.Config{URI: dsn}

	testDB := dbName(name)
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(cfg.URI)
	require.NoError(t, err)
	poolConfig.ConnConfig.Database = testDB

	tracer := &tracelog.TraceLog{
		Logger:   &testLogger{t: t},
		LogLevel: tracelog.LogLevelDebug,
	}
	poolConfig.ConnConfig.Tracer = tracer

	conn, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	managementConnCfg := poolConfig.ConnConfig.Copy()
	err = dbCreate(ctx, managementConnCfg, testDB)
	require.NoError(t, err)

	db := &postgres.DB{Pool: conn}
	err = db.Ping(ctx)
	require.NoError(t, err)

	migrationConn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer migrationConn.Release()

	err = postgres.MigrateUp(ctx, migrationConn.Conn(), migrations.MigrationsFiles)
	require.NoError(t, err)

	teardown := func() {
		db.Close()

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

func CountRows(t *testing.T, table string, db postgres.Querier) int {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	err := db.QueryRow(context.Background(), query).Scan(&count)
	require.NoError(t, err)
	return count
}

// NowLocalTime returns time with fixed zone
// Using this helper in test fixtures allows to compare structs after fetching from storage
// Avoids lib/pq issue with `+0000` vs stdlib.Time `UTC`
func NowLocalTime() time.Time {
	return time.Now().Round(time.Microsecond).Local()
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Log(_ context.Context, _ tracelog.LogLevel, msg string, data map[string]interface{}) {
	if msg != "Query" {
		return
	}

	sql, ok := data["sql"]
	if ok {
		l.t.Logf("%s %s %s", msg, sql, data["args"])
	} else {
		l.t.Logf("%s", msg)
	}
}
