package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/agalitsyn/goth/pkg/flagtools"
	"github.com/agalitsyn/goth/pkg/secret"
	"github.com/agalitsyn/goth/pkg/slogtools"
	"github.com/agalitsyn/goth/pkg/version"
)

type Config struct {
	Debug bool

	Log struct {
		Level slog.Level
	}

	HTTP struct {
		Addr               string
		ShutdownTimeoutSec time.Duration
	}

	Postgres struct {
		ConnectionString secret.String
	}
}

func (c Config) String() string {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(0)
	}
	return string(b)
}

func ParseFlags() Config {
	var cfg Config

	printVersion := flag.Bool("version", false, "Show application version.")
	logLevel := flag.String("log-level", "info", "Log level (debug|info|warn|error).")
	pgDSN := flag.String("postgres-dsn", "postgres://postgres:postgres@localhost:5432/postgres", "PostgreSQL connection string.")
	flag.StringVar(&cfg.HTTP.Addr, "http-addr", "localhost:8080", "HTTP service address.")
	httpShutdownTimeoutSec := flag.Int("http-shutdown", 10, "HTTP service graceful shutdown timeout (sec).")

	flagtools.Prefix = ""
	flagtools.Parse()
	flag.Parse()

	cfg.Postgres.ConnectionString = secret.NewString(*pgDSN)
	*pgDSN = ""
	slogLevel := slogtools.ParseLogLevel(*logLevel)
	cfg.Log.Level = slogLevel
	cfg.HTTP.ShutdownTimeoutSec = time.Duration(*httpShutdownTimeoutSec) * time.Second

	if slogLevel == slog.LevelDebug {
		cfg.Debug = true
	}

	if *printVersion {
		fmt.Fprintln(os.Stdout, version.String())
		os.Exit(0)
	}

	return cfg
}
