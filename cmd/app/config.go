package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/agalitsyn/goth/pkg/flagtools"
	"github.com/agalitsyn/goth/pkg/secret"
	"github.com/agalitsyn/goth/pkg/slogtools"
	"github.com/agalitsyn/goth/pkg/version"
)

const EnvPrefix = "GOTH"

type Config struct {
	Debug bool

	Log struct {
		Level slog.Level
	}

	HTTP struct {
		Addr               string
		ShutdownTimeoutSec time.Duration

		CorsAllowedOrigins []string
		CorsAllowedHeaders []string
		CorsExposedHeaders []string
	}

	Postgres struct {
		ConnectionString secret.String
		Host             string
		Port             string
		User             string
		Pass             secret.String
		DB               string
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

	logLevel := flag.String("log-level", "info", "Log level (debug | info | warn | error).")

	pgDSN := flag.String(
		"postgres-uri",
		"",
		"PostgreSQL connection string (if empty app will use host, port, user, pass, db settings).",
	)

	flag.StringVar(&cfg.Postgres.Host, "postgres-host", "localhost", "PostgreSQL host.")
	flag.StringVar(&cfg.Postgres.Port, "postgres-port", "5432", "PostgreSQL port.")
	flag.StringVar(&cfg.Postgres.User, "postgres-user", "postgres", "PostgreSQL user.")
	pgPass := flag.String("postgres-pass", "postgres", "PostgreSQL password.")
	flag.StringVar(&cfg.Postgres.DB, "postgres-db", "postgres", "PostgreSQL database.")

	flag.StringVar(&cfg.HTTP.Addr, "http-addr", "localhost:8080", "HTTP service address.")
	httpShutdownTimeoutSec := flag.Int("http-shutdown", 10, "HTTP service graceful shutdown timeout (sec).")
	corsAllowedOrigins := flag.String(
		"http-cors-allowed-origins",
		"*",
		"The list of origins a cross-domain request can be executed from.",
	)
	corsAllowedHeaders := flag.String(
		"http-cors-allowed-headers",
		"*",
		"The list of non simple headers the client is allowed to use with cross-domain requests.",
	)
	corsExposedHeaders := flag.String(
		"http-cors-exposed-headers",
		"*",
		"The list which indicates which headers are safe to expose.",
	)

	flagtools.Prefix = EnvPrefix
	flagtools.Parse()
	flag.Parse()

	if *pgDSN != "" {
		cfg.Postgres.ConnectionString = secret.NewString(*pgDSN)
		*pgDSN = ""
	} else {
		cfg.Postgres.Pass = secret.NewString(*pgPass)
		*pgPass = ""
	}

	slogLevel := slogtools.ParseLogLevel(*logLevel)

	cfg.Log.Level = slogLevel
	cfg.HTTP.ShutdownTimeoutSec = time.Duration(*httpShutdownTimeoutSec) * time.Second
	cfg.HTTP.CorsAllowedOrigins = strings.Split(*corsAllowedOrigins, ",")
	cfg.HTTP.CorsAllowedHeaders = strings.Split(*corsAllowedHeaders, ",")
	cfg.HTTP.CorsExposedHeaders = strings.Split(*corsExposedHeaders, ",")

	if slogLevel == slog.LevelDebug {
		cfg.Debug = true
	}

	if *printVersion {
		fmt.Fprintln(os.Stdout, version.String())
		os.Exit(0)
	}

	return cfg
}
