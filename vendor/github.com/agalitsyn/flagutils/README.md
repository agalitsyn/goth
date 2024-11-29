# flagutils

## Example

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/agalitsyn/flagutils"
	"github.com/agalitsyn/postgres"
	"github.com/agalitsyn/secret"
	"github.com/agalitsyn/slogutils"
	"github.com/agalitsyn/version"
)

const EnvPrefix = "MY_API"

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

	logLevel := flag.String("log-level", "info", "Log level (debug | info | warn | error).")

	pgDSN := flag.String(
		"postgres-uri",
		"",
		"PostgreSQL connection string (if empty app will use host, port, user, pass, db settings).",
	)
	flag.StringVar(&cfg.Postgres.Host, "postgres-host", "", "PostgreSQL host.")
	flag.StringVar(&cfg.Postgres.Port, "postgres-port", "", "PostgreSQL port.")
	flag.StringVar(&cfg.Postgres.User, "postgres-user", "", "PostgreSQL user.")
	pgPass := flag.String("postgres-pass", "", "PostgreSQL password.")
	flag.StringVar(&cfg.Postgres.DB, "postgres-db", "", "PostgreSQL database.")

	flag.StringVar(&cfg.HTTP.Addr, "http-addr", "localhost:8888", "HTTP service address.")
	httpShutdownTimeoutSec := flag.Int("http-shutdown", 10, "HTTP service graceful shutdown timeout (sec).")

	printVersion := flag.Bool("version", false, "Show application version.")

	flagutils.Prefix = EnvPrefix
	flagutils.Parse()
	flag.Parse()

	if *pgDSN != "" {
		cfg.Postgres.ConnectionString = secret.NewString(*pgDSN)
		*pgDSN = ""
	} else if *pgPass != "" {
		cfg.Postgres.Pass = secret.NewString(*pgPass)
		*pgPass = ""
	}

	cfg.HTTP.ShutdownTimeoutSec = time.Duration(*httpShutdownTimeoutSec) * time.Second

	slogLevel := slogutils.ParseLogLevel(*logLevel)
	cfg.Log.Level = slogLevel
	if slogLevel == slog.LevelDebug {
		cfg.Debug = true
	}

	if *printVersion {
		fmt.Fprintln(os.Stdout, version.String())
		os.Exit(0)
	}

	return cfg
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := ParseFlags()
	slogutils.SetupGlobalLogger(cfg.Log.Level, os.Stdout)

	if cfg.Debug {
		slog.Debug("running with config")
		fmt.Fprintln(os.Stdout, cfg.String())
	}

	pgCfg := postgres.Config{
		URI:            cfg.Postgres.ConnectionString.Unmask(),
		Host:           cfg.Postgres.Host,
		Port:           cfg.Postgres.Port,
		User:           cfg.Postgres.User,
		Pass:           cfg.Postgres.Pass.Unmask(),
		DB:             cfg.Postgres.DB,
		TracerLogLevel: "error",
	}
	if cfg.Debug {
		pgCfg.TracerLogLevel = "debug"
	}
	pg, err := postgres.New(ctx, pgCfg)
	if err != nil {
		slogutils.Fatal("could not create postgres client", "error", err)
	}
	defer pg.Close()

	if err = pg.RetryConnect(ctx); err != nil {
		slogutils.Fatal("could not connect to postgres", "error", err)
	}

	router := http.NewServeMux()
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
		w.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		// make a new context for the Shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeoutSec)
		defer cancel()

		slog.Info("gracefully shutting down http server")
		if err = httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutting down http server", "error", err)
		}
	}()
	slog.Info("starting http server", "addr", httpServer.Addr)
	if err = httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "error", err)
	}
}
```
