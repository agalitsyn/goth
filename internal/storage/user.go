package storage

import (
	"context"

	"github.com/google/uuid"

	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/pkg/postgres/pagination"
)

type UserFilterParams struct {
	pagination.Pagination
}

type UserSessionsFilterParams struct {
	UserID    int64
	IsExpired bool
}

type UserStorage interface {
	FetchUserByID(ctx context.Context, id int64) (*model.User, error)
	FetchUserByLogin(ctx context.Context, login string) (*model.User, error)
	FetchUserPassword(ctx context.Context, user *model.User) error
	FilterUsers(ctx context.Context, filter UserFilterParams) ([]model.User, error)
	CreateUser(ctx context.Context, user *model.User) error

	UpdateUserSession(ctx context.Context, session *model.UserSession) error
	FilterUserSessions(ctx context.Context, filter UserSessionsFilterParams) ([]model.UserSession, error)
	FetchUserSession(ctx context.Context, uuid uuid.UUID) (*model.UserSession, error)
	CreateUserSession(ctx context.Context, session *model.UserSession) error
	DeleteUserSessions(ctx context.Context, sessions []model.UserSession) error
}
