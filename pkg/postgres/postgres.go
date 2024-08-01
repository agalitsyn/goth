package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"

	"github.com/agalitsyn/goth/pkg/secret"
)

const (
	DriverName             = "pgx"
	defaultMaxConnLifetime = time.Hour
	defaultMaxOpenConns    = 4
)

type Config struct {
	URI             secret.String
	MaxConnLifetime time.Duration
	ConnectInterval time.Duration
	ConnectTimeout  time.Duration
}

func (c *Config) CheckAndSetDefaults() error {
	if c.URI.Unmask() == "" {
		return errors.New("URI is required")
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = defaultMaxConnLifetime
	}
	if c.ConnectInterval == 0 {
		c.ConnectInterval = 1 * time.Second
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 10 * time.Second
	}
	return nil
}

type DB struct {
	Cfg     Config
	Session *sqlx.DB
}

func New(cfg Config) (*DB, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	pgxCfg, err := pgxpool.ParseConfig(cfg.URI.Unmask())
	if err != nil {
		return nil, fmt.Errorf("dsn parse error: %s", err)
	}
	// Note: in case of using pgbouncer or pgpool
	//pgxCfg.ConnConfig.PreferSimpleProtocol = true
	//pgxCfg.ConnConfig.BuildStatementCache = func(conn *pgconn.PgConn) stmtcache.Cache {
	//	return stmtcache.New(conn, stmtcache.ModeDescribe, 512)
	//}

	db, err := sql.Open(
		DriverName,
		stdlib.RegisterConnConfig(pgxCfg.ConnConfig),
	)
	if err != nil {
		return nil, fmt.Errorf("driver open error: %w", err)
	}
	db.SetConnMaxLifetime(cfg.MaxConnLifetime)

	dbx := sqlx.NewDb(db, DriverName)
	session := &DB{
		Session: dbx,
		Cfg:     cfg,
	}
	return session, nil
}

func (d *DB) RetryConnect(ctx context.Context) error {
	connectFunc := func(ctx context.Context) error {
		if err := d.PingContext(ctx); err != nil {
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

func (d *DB) PingContext(ctx context.Context) error {
	return d.Session.PingContext(ctx)
}

func (d *DB) Close() error {
	return d.Session.Close()
}

func (d *DB) InTx(ctx context.Context, f func(tx *sqlx.Tx) error) error {
	tx, finishTx, err := d.beginTx(ctx)
	if err != nil {
		return err
	}
	return finishTx(f(tx))
}

func (d *DB) beginTx(ctx context.Context) (*sqlx.Tx, func(error) error, error) {
	tx, err := d.Session.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, nil, err
	}

	// it commits or rollbacks initialized transaction
	finishTx := func(err error) error {
		if err != nil {
			e := tx.Rollback()
			if e != nil && e != sql.ErrTxDone {
				err = fmt.Errorf("failed to rollback transaction (err=%s) after error %s", e, err)
			}
			return err
		}

		e := tx.Commit()
		if e == sql.ErrTxDone {
			return fmt.Errorf("failed to commit rollbacked transaction (timeout): %w", sql.ErrTxDone)
		}
		if e != nil {
			return fmt.Errorf("failed to commit transaction: %s", e.Error())
		}

		return nil
	}

	return tx, finishTx, err
}

func IsDuplicateKeyError(err error) bool {
	pgErr := new(pgconn.PgError)
	if errors.As(err, &pgErr) {
		// unique_violation = 23505
		if pgErr.Code == "23505" {
			return true
		}
	}
	return false
}
