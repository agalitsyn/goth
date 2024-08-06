package main

import (
	"github.com/spf13/cobra"
)

func NewAdminGroup(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin [action]",
		Short: "Admin commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := rootCmd.PersistentPreRunE(cmd, args); err != nil {
				return err
			}

			pgConnURL, err := cmd.Flags().GetString("postgres-uri")
			if err != nil {
				return err
			}

			pg, err := setupPostgres(cmd.Context(), pgConnURL, d.isDebug())
			if err != nil {
				return err
			}

			d.db = pg
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if d.db != nil {
				d.db.Close()
			}
			return nil
		},
	}
	cmd.Flags().SortFlags = false

	cmd.PersistentFlags().String(
		"postgres-uri",
		"",
		"PostgreSQL connection string",
	)

	cmd.AddCommand(NewDBGroup(d))
	cmd.AddCommand(NewUserGroup(d))

	for _, c := range cmd.Commands() {
		c.SilenceErrors = true
		c.SilenceUsage = true
		c.Flags().SortFlags = false
	}

	return cmd
}
