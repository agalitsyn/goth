package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/internal/storage"
)

type SessionAuthenticatorConfig struct {
	LoginRedirectURL  string
	PageRedirectURL   string
	SessionMaxAgeInDB time.Duration
	CookieName        string
	CookieMaxAge      int
	CookieSecure      bool
}

type SessionAuthenticator struct {
	cfg                SessionAuthenticatorConfig
	userValidationFunc model.UserValidationFunc
	userStorage        storage.UserStorage
}

func NewSessionAuthenticator(
	cfg SessionAuthenticatorConfig,
	userStorage storage.UserStorage,
	userValidationFunc model.UserValidationFunc,
) *SessionAuthenticator {
	return &SessionAuthenticator{
		cfg:                cfg,
		userStorage:        userStorage,
		userValidationFunc: userValidationFunc,
	}
}

var (
	ErrLoginPassword = errors.New("auth: invalid login or password")
	ErrInvalidUser   = errors.New("auth: invalid user")
	ErrForbidden     = errors.New("auth: user dont have enough permissions")
	ErrInternal      = errors.New("auth: internal error")
)

func (s *SessionAuthenticator) CreateSession(ctx context.Context, login string, password string) (*model.UserSession, error) {
	user, err := s.userStorage.FetchUserByLogin(ctx, login)
	if err != nil {
		slog.Error("could not fetch user", "error", err)
		// mask not found error as invalid login or password
		if errors.Is(err, storage.ErrNotFound) {
			return nil, ErrLoginPassword
		}
		return nil, ErrInternal
	}
	if err := s.userValidationFunc(user); err != nil {
		slog.Error("invalid user", "user_id", user.ID, "error", err)
		return nil, ErrForbidden
	}

	if err := s.userStorage.FetchUserPassword(ctx, user); err != nil {
		slog.Error("could not fetch user password", "error", err)
		return nil, ErrInternal
	}

	if err := model.CompareUserPassword([]byte(user.HashedPassword), password); err != nil {
		slog.Error("user password input and hash mismatch", "error", err)
		return nil, ErrLoginPassword
	}

	// delete expired sessions
	// if it fails allow to proceed anyway, will try to delete next time
	filter := storage.UserSessionsFilterParams{
		UserID:    user.ID,
		IsExpired: true,
	}
	sessions, err := s.userStorage.FilterUserSessions(ctx, filter)
	if err != nil {
		slog.Error("could not filter user sessions", "error", err)
	}
	if len(sessions) > 0 {
		if err := s.userStorage.DeleteUserSessions(ctx, sessions); err != nil {
			slog.Error("could not delete user sessions", "error", err)
		}
	}

	session := &model.UserSession{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.cfg.SessionMaxAgeInDB),
	}
	if err := s.userStorage.CreateUserSession(ctx, session); err != nil {
		slog.Error("could not create user session", "error", err)
		return nil, ErrInternal
	}
	return session, nil
}

func (s *SessionAuthenticator) LoginRequiredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.cfg.CookieName)
		if err != nil {
			slog.Error("could not get session cookie", "error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		sid := uuid.MustParse(cookie.Value)
		session, err := s.userStorage.FetchUserSession(r.Context(), sid)
		if err != nil {
			// delete session cookie if session is not found
			if errors.Is(err, storage.ErrNotFound) {
				cookie := s.DeletionSessionCookie()
				http.SetCookie(w, cookie)
			} else {
				slog.Error("could not fetch user session", "session_id", sid, "error", err)
			}

			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if session.ExpiresAt.Before(time.Now()) {
			slog.Warn("user session expired", "session_id", sid, "user_id", session.UserID)
			cookie := s.DeletionSessionCookie()
			http.SetCookie(w, cookie)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		user, err := s.userStorage.FetchUserByID(r.Context(), session.UserID)
		if err != nil {
			// delete session cookie if user is not found
			if errors.Is(err, storage.ErrNotFound) {
				cookie := s.DeletionSessionCookie()
				http.SetCookie(w, cookie)
			} else {
				slog.Error("could not fetch user", "user_id", session.UserID, "error", err)
			}

			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if err := s.userValidationFunc(user); err != nil {
			slog.Error("invalid user", "user_id", user.ID, "error", err)
			cookie := s.DeletionSessionCookie()
			http.SetCookie(w, cookie)
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// Redirect to non-login page if user is already on login page
		if r.URL.Path == s.cfg.LoginRedirectURL {
			http.Redirect(w, r, s.cfg.PageRedirectURL, http.StatusMovedPermanently)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *SessionAuthenticator) MakeSessionCookie(value fmt.Stringer) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    value.String(),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   s.cfg.CookieMaxAge,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.cfg.CookieSecure,
	}
}

func (s *SessionAuthenticator) DeletionSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:   s.cfg.CookieName,
		Path:   "/",
		MaxAge: -1,
	}
}

type contextKey string

const userContextKey contextKey = "user"

func UserFromContext(ctx context.Context) (*model.User, error) {
	user, ok := ctx.Value(userContextKey).(*model.User)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}
	return user, nil
}

// MustUserFromContext panics if not user in context.
// Designed to be used in handlers where user is guaranteed to be in context.
// Protects from developer mistakes in router config.
func MustUserFromContext(ctx context.Context) *model.User {
	user, err := UserFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return user
}
