package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/internal/storage"
	"github.com/agalitsyn/goth/pkg/postgres"
)

type UserStorage struct {
	db *postgres.DB
}

func NewUserStorage(db *postgres.DB) *UserStorage {
	return &UserStorage{db: db}
}

func (s *UserStorage) FilterUsers(ctx context.Context, params storage.UserFilterParams) ([]model.User, error) {
	q := sq.Select("id", "login", "is_active").From("users").
		Offset(params.Offset)
	for _, sort := range params.Sort {
		q = q.OrderBy(fmt.Sprintf("%s %s", sort.By, sort.Order))
	}
	if params.Limit > 0 {
		q = q.Limit(params.Limit)
	}

	query, args := q.PlaceholderFormat(sq.Dollar).MustSql()
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not perform query: %w", err)
	}
	defer rows.Close()

	var res []model.User
	for rows.Next() {
		var user model.User
		err = rows.Scan(
			&user.ID,
			&user.Login,
			&user.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}
		res = append(res, user)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("could not iterate rows: %w", err)
	}

	return res, nil
}

func (s *UserStorage) FetchUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	// language=PostgreSQL
	q := `SELECT id, login, is_active FROM users WHERE login = $1`
	err := s.db.QueryRow(ctx, q, login).Scan(
		&user.ID,
		&user.Login,
		&user.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("could not perform query: %w", err)
	}
	return &user, nil
}

func (s *UserStorage) FetchUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	// language=PostgreSQL
	q := `SELECT id, login, is_active FROM users WHERE id = $1`
	err := s.db.QueryRow(ctx, q, id).Scan(
		&user.ID,
		&user.Login,
		&user.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("could not perform query: %w", err)
	}
	return &user, nil
}

func (s *UserStorage) FetchUserPassword(ctx context.Context, user *model.User) error {
	if user.ID == 0 {
		return fmt.Errorf("cannot fetch user password with empty user id")
	}

	// language=PostgreSQL
	q := `SELECT hashed_password FROM users WHERE id = $1`
	err := s.db.QueryRow(ctx, q, user.ID).Scan(&user.HashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("could not perform query: %w", err)
	}
	return nil
}

func (s *UserStorage) CreateUser(ctx context.Context, user *model.User) error {
	// language=PostgreSQL
	q := `INSERT INTO users (login, hashed_password, is_active)
VALUES ($1, $2, $3)
ON CONFLICT (login) DO UPDATE SET
	hashed_password = excluded.hashed_password,
	is_active = excluded.is_active
RETURNING id`
	err := s.db.QueryRow(ctx, q, user.Login, user.HashedPassword, user.IsActive).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("could not perform query: %w", err)
	}
	return nil
}

func (s *UserStorage) FilterUserSessions(
	ctx context.Context,
	params storage.UserSessionsFilterParams,
) ([]model.UserSession, error) {
	q := sq.Select("uuid", "user_id", "expires_at").
		From("user_sessions")

	if params.UserID != 0 {
		q = q.Where("user_id = ?", params.UserID)
	}
	if params.IsExpired {
		q = q.Where("expires_at < NOW()")
	}

	query, args := q.PlaceholderFormat(sq.Dollar).MustSql()
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not perform query: %w", err)
	}
	defer rows.Close()

	var sessions []model.UserSession
	for rows.Next() {
		var res model.UserSession
		err = rows.Scan(
			&res.UUID,
			&res.UserID,
			&res.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}
		sessions = append(sessions, res)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("could not iterate rows: %w", err)
	}

	return sessions, nil
}

func (s *UserStorage) FetchUserSession(ctx context.Context, uuid uuid.UUID) (*model.UserSession, error) {
	// language=PostgreSQL
	q := `SELECT uuid, user_id, expires_at FROM user_sessions WHERE uuid = $1`
	var session model.UserSession
	err := s.db.QueryRow(ctx, q, uuid).Scan(
		&session.UUID,
		&session.UserID,
		&session.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("could not perform query: %w", err)
	}
	return &session, nil
}

func (s *UserStorage) CreateUserSession(ctx context.Context, session *model.UserSession) error {
	// language=PostgreSQL
	q := `INSERT INTO user_sessions (user_id, expires_at) VALUES ($1, $2) RETURNING uuid`
	err := s.db.QueryRow(ctx, q, session.UserID, session.ExpiresAt).Scan(&session.UUID)
	if err != nil {
		return fmt.Errorf("could not perform query: %w", err)
	}
	return nil
}

func (s *UserStorage) DeleteUserSessions(ctx context.Context, sessions []model.UserSession) error {
	if len(sessions) == 0 {
		return nil
	}

	// language=PostgreSQL
	q := `DELETE FROM user_sessions WHERE uuid = ANY($1)`
	ids := make([]uuid.UUID, len(sessions))
	for i, session := range sessions {
		ids[i] = session.UUID
	}
	_, err := s.db.Exec(ctx, q, ids)
	if err != nil {
		return fmt.Errorf("could not perform query: %w", err)
	}
	return nil
}

func (s *UserStorage) UpdateUserSession(ctx context.Context, session *model.UserSession) error {
	// language=PostgreSQL
	q := `UPDATE user_sessions SET expires_at = $1 WHERE uuid = $2`
	_, err := s.db.Exec(ctx, q, session.ExpiresAt, session.UUID)
	if err != nil {
		return fmt.Errorf("could not perform query: %w", err)
	}
	return nil
}
