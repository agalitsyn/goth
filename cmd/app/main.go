package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/agalitsyn/goth/pkg/postgres"
	"github.com/agalitsyn/goth/pkg/slogtools"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := ParseFlags()
	slogtools.SetupGlobalLogger(cfg.Log.Level, os.Stdout)

	if cfg.Debug {
		slog.Debug("running with config")
		fmt.Fprintln(os.Stdout, cfg.String())
	}

	pg, err := postgres.New(postgres.Config{URI: cfg.Postgres.ConnectionString})
	if err != nil {
		slogtools.Fatal("could not create postgres client", "error", err)
	}
	defer pg.Close()

	if err = pg.RetryConnect(ctx); err != nil {
		slogtools.Fatal("could not connect to postgres", "error", err)
	}

	router, err := MakeRouter()
	if err != nil {
		slog.Error("could not create router", "error", err)
		return
	}

	httpServer := &http.Server{Addr: cfg.HTTP.Addr, Handler: router}
	go func() {
		<-ctx.Done()
		// make a new context for the Shutdown
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(ctx, cfg.HTTP.ShutdownTimeoutSec)
		defer cancel()

		slog.Info("gracefully shutting down http server")
		if err = httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutting down http server", "error", err)
		}
	}()
	slog.Info("starting http server", "addr", httpServer.Addr)
	if err = httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "error", err)
	}
}
