package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"
	pglog "github.com/mcosta74/pgx-slog"

	"github.com/agalitsyn/secret"
)

// Querier is the interface postgres package uses to access the database. It is satisfied by *pgx.Conn, pgx.Tx, *pgxpool.Pool, etc.
type Querier interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) (br pgx.BatchResults)
}

type DB struct {
	*pgxpool.Pool
}

func New(ctx context.Context, cfg Config) (*DB, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URI.Unmask())
	if err != nil {
		return nil, fmt.Errorf("could not parse connection string: %w", err)
	}

	tracer := &tracelog.TraceLog{
		Logger:   pglog.NewLogger(slog.Default()),
		LogLevel: cfg.tracerLogLevelParsed,
	}
	poolConfig.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create connection pool: %w", err)
	}
	return &DB{Pool: pool}, nil
}

type Config struct {
	URI  secret.String
	Host string
	Port string
	User string
	Pass secret.String
	DB   string

	TracerLogLevel       string
	tracerLogLevelParsed tracelog.LogLevel

	MaxConnLifetime time.Duration
}

func (c *Config) CheckAndSetDefaults() error {
	if c.URI.Unmask() == "" {
		if c.Host == "" || c.Port == "" || c.User == "" || c.Pass.Unmask() == "" || c.DB == "" {
			return errors.New("URI or Host, Port, User, Pass, DB is required")
		}

		hostPort := c.Host
		if c.Port != "" {
			hostPort = net.JoinHostPort(c.Host, c.Port)
		}
		uri := url.URL{
			Scheme: "postgres",
			Host:   hostPort,
			Path:   fmt.Sprintf("/%s", c.DB),
		}
		uri.User = url.UserPassword(c.User, c.Pass.Unmask())
		c.URI = secret.NewString(uri.String())
	}

	if c.TracerLogLevel == "" {
		c.tracerLogLevelParsed = tracelog.LogLevelError
	} else {
		lvl, err := tracelog.LogLevelFromString(c.TracerLogLevel)
		if err != nil {
			return fmt.Errorf("invalid tracer log level: %s", err)
		}
		c.tracerLogLevelParsed = lvl
	}

	return nil
}

func (d *DB) RetryConnect(ctx context.Context) error {
	connectFunc := func(ctx context.Context) error {
		if err := d.Ping(ctx); err != nil {
			return err
		}
		return nil
	}

	how := retry.How{
		strategy.Limit(10),
		strategy.Backoff(func(attempt uint) time.Duration {
			backoff := time.Duration(attempt) * 100 * time.Millisecond
			slog.Warn("establishing db connection", "attempt", attempt)
			return backoff
		}),
	}
	if err := retry.Do(ctx, connectFunc, how...); err != nil {
		return err
	}
	return nil
}
