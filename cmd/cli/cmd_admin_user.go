package main

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/internal/storage"
	"github.com/agalitsyn/goth/internal/storage/postgres"
)

func NewUserGroup(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user [action]",
		Short: "User commands",
	}
	cmd.Flags().SortFlags = false
	cmd.SilenceErrors = true

	cmd.AddCommand(NewUserCreateCommand(d))
	cmd.AddCommand(NewUserSessionGroup(d))

	for _, c := range cmd.Commands() {
		c.SilenceErrors = true
		c.SilenceUsage = true
		c.Flags().SortFlags = false
	}

	return cmd
}

type UserCreateOptions struct {
	Login    string
	Password string
}

func NewUserCreateCommand(d *deps) *cobra.Command {
	var opts UserCreateOptions
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create or update user",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("run user create", "args", args, "opts", fmt.Sprintf("%+v", opts))

			passwordHash, err := model.HashUserPassword(opts.Password)
			if err != nil {
				return err
			}
			user := &model.User{
				Login:          opts.Login,
				HashedPassword: string(passwordHash),
				IsActive:       true,
			}

			userStorage := postgres.NewUserStorage(d.db)
			if err := userStorage.CreateUser(cmd.Context(), user); err != nil {
				return err
			}

			fmt.Printf("user created: id=%d\n", user.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(
		&opts.Login,
		"login",
		"",
		"User login",
	)
	cmd.MarkFlagRequired("login")

	cmd.Flags().StringVar(
		&opts.Password,
		"password",
		"",
		"User password",
	)
	cmd.MarkFlagRequired("password")

	return cmd
}

func NewUserSessionGroup(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session [action]",
		Short: "User session commands",
	}
	cmd.Flags().SortFlags = false
	cmd.SilenceErrors = true

	cmd.AddCommand(NewUserSessionDeleteCommand(d))

	for _, c := range cmd.Commands() {
		c.SilenceErrors = true
		c.SilenceUsage = true
		c.Flags().SortFlags = false
	}

	return cmd
}

type UserSessionDeleteOptions struct {
	Login     string
	IsExpired bool
}

func NewUserSessionDeleteCommand(d *deps) *cobra.Command {
	var opts UserSessionDeleteOptions
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete user sessions",
		Example: `
session delete --login=foo - delete expired sessions for user
session delete --login=foo --expired=false - delete all sessions for user (DANGER!)
session delete --expired=false - delete all sessions (DANGER!)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("run session delete", "args", args, "opts", fmt.Sprintf("%+v", opts))

			msg := fmt.Sprintf("user sessions deleted: expired only=%v", opts.IsExpired)

			userStorage := postgres.NewUserStorage(d.db)

			filter := storage.UserSessionsFilterParams{
				IsExpired: opts.IsExpired,
			}

			if opts.Login != "" {
				user, err := userStorage.FetchUserByLogin(cmd.Context(), opts.Login)
				if err != nil {
					return err
				}
				filter.UserID = user.ID
				msg += fmt.Sprintf(", login=%s, id=%d", opts.Login, user.ID)
			}

			sessions, err := userStorage.FilterUserSessions(cmd.Context(), filter)
			if err != nil {
				return err
			}

			if err := userStorage.DeleteUserSessions(cmd.Context(), sessions); err != nil {
				return err
			}

			fmt.Println(msg)
			return nil
		},
	}
	cmd.Flags().StringVar(
		&opts.Login,
		"login",
		"",
		"User login",
	)

	cmd.Flags().BoolVar(
		&opts.IsExpired,
		"expired",
		true,
		"Only expired sessions",
	)

	return cmd
}
