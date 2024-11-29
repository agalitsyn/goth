package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/agalitsyn/goth/pkg/version"
	"github.com/agalitsyn/postgres"
	"github.com/agalitsyn/secret"
	"github.com/agalitsyn/slogutils"
)

var (
	flagLogLevel string

	d deps

	rootCmd = &cobra.Command{
		Use:   "cli",
		Short: "",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			setupLogger()

			var err error
			d, err = initDeps()

			return err
		},
	}
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cobra.EnableCommandSorting = false
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false

	allowedLogLevels := []string{
		slog.LevelDebug.String(),
		slog.LevelInfo.String(),
		slog.LevelWarn.String(),
		slog.LevelError.String(),
	}
	rootCmd.PersistentFlags().StringVarP(&flagLogLevel,
		"log-level",
		"l",
		slog.LevelInfo.String(),
		fmt.Sprintf("Log level (%s)", strings.Join(allowedLogLevels, " | ")),
	)

	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewAdminGroup(&d))

	cobra.CheckErr(rootCmd.ExecuteContext(ctx))
}

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(version.String())
			return nil
		},
	}
	return cmd
}

type deps struct {
	logLevel slog.Level
	db       *postgres.DB
}

func initDeps() (deps, error) {
	d := deps{
		logLevel: slogutils.ParseLogLevel(flagLogLevel),
	}
	return d, nil
}

func (d deps) isDebug() bool {
	return d.logLevel == slog.LevelDebug
}

func setupLogger() {
	v, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
		flagLogLevel = v
	}

	lvl := slogutils.ParseLogLevel(flagLogLevel)
	w := os.Stdout
	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      lvl,
			TimeFormat: time.DateTime,
			NoColor:    !isatty.IsTerminal(w.Fd()),
		}),
	)
	slog.SetDefault(logger)
}

func setupPostgres(ctx context.Context, connString string, debug bool) (*postgres.DB, error) {
	v, ok := os.LookupEnv("POSTGRES_URI")
	if ok {
		connString = v
	}

	pgCfg := postgres.Config{URI: secret.NewString(connString)}
	if debug {
		pgCfg.TracerLogLevel = "debug"
	}

	pg, err := postgres.New(ctx, pgCfg)
	if err != nil {
		return nil, fmt.Errorf("could not create postgres client: %w", err)
	}

	if err = pg.RetryConnect(ctx); err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	return pg, nil
}
