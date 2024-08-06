package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/agalitsyn/goth/pkg/postgres"

	"github.com/agalitsyn/goth/migrations"
)

func NewDBGroup(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db [action]",
		Short: "Database commands",
	}
	cmd.Flags().SortFlags = false
	cmd.SilenceErrors = true

	cmd.AddCommand(NewMigrateCommand(d))

	for _, c := range cmd.Commands() {
		c.SilenceErrors = true
		c.SilenceUsage = true
		c.Flags().SortFlags = false
	}

	return cmd
}

func NewMigrateCommand(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("run migrate", "args", args)

			conn, err := d.db.Pool.Acquire(cmd.Context())
			if err != nil {
				return err
			}
			defer conn.Release()

			err = postgres.MigrateUp(cmd.Context(), conn.Conn(), migrations.MigrationsFiles)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
