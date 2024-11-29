package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/go-chi/cors"

	"github.com/agalitsyn/goth/cmd/admin/controller"
	"github.com/agalitsyn/goth/cmd/admin/renderer"
	"github.com/agalitsyn/goth/internal/auth"
	postgresStorage "github.com/agalitsyn/goth/internal/storage/postgres"
	"github.com/agalitsyn/goth/pkg/httptools"
	"github.com/agalitsyn/postgres"
	"github.com/agalitsyn/slogutils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := ParseFlags()
	slogutils.SetupGlobalLogger(cfg.Log.Level, os.Stdout)

	if cfg.Debug {
		slog.Debug("running with config")
		fmt.Fprintln(os.Stdout, cfg.String())
	}

	templates, err := httptools.NewTemplateCache(EmbedFiles, "templates", templateFuncs())
	if err != nil {
		slogutils.Fatal("could not load templates", "error", err)
	}

	htmlRenderer := renderer.NewHTMLRenderer(httptools.NewTemplateRenderer(templates))
	if cfg.Debug {
		htmlRenderer.Debug = true

		slog.Debug("loaded templates")
		names := make([]string, 0, len(templates))
		for name := range templates {
			names = append(names, name)
		}
		sort.Strings(names)
		fmt.Fprintln(os.Stdout, names)
	}

	pgCfg := postgres.Config{
		URI:            cfg.Postgres.ConnectionString,
		Host:           cfg.Postgres.Host,
		Port:           cfg.Postgres.Port,
		User:           cfg.Postgres.User,
		Pass:           cfg.Postgres.Pass,
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

	userStorage := postgresStorage.NewUserStorage(pg)

	authenticatorCfg := auth.SessionAuthenticatorConfig{
		LoginRedirectURL:  "/login",
		PageRedirectURL:   "/",
		SessionMaxAgeInDB: time.Hour * 24 * 31, // 1 month
		CookieName:        "admin_session_id",
		CookieMaxAge:      60 * 60 * 24 * 365, // 1 year in seconds
		// TODO: configure if https
		CookieSecure: false,
	}
	authenticator := auth.NewSessionAuthenticator(authenticatorCfg, userStorage, checkUserIsActive)

	userCtrl := controller.NewUserController(htmlRenderer, authenticator, userStorage)

	corsCfg := cors.Options{
		AllowedOrigins:   cfg.HTTP.CorsAllowedOrigins,
		AllowedHeaders:   cfg.HTTP.CorsAllowedHeaders,
		ExposedHeaders:   cfg.HTTP.CorsExposedHeaders,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowCredentials: true,
	}
	corsMiddleware := cors.New(corsCfg)

	router, err := NewRouter(
		corsMiddleware.Handler,
		authenticator.LoginRequiredMiddleware,
		htmlRenderer,
		userCtrl,
	)
	if err != nil {
		slog.Error("could not create router", "error", err)
		return
	}

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
