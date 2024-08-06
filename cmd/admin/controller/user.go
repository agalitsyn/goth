package controller

import (
	"errors"
	"net/http"

	"github.com/agalitsyn/goth/cmd/admin/renderer"
	"github.com/agalitsyn/goth/internal/auth"
	"github.com/agalitsyn/goth/internal/storage"
	"github.com/agalitsyn/goth/pkg/httptools/validator"
)

type UserController struct {
	authenticator *auth.SessionAuthenticator
	userStorage   storage.UserStorage

	*renderer.HTMLRenderer
}

func NewUserController(r *renderer.HTMLRenderer, authenticator *auth.SessionAuthenticator, userStorage storage.UserStorage) *UserController {
	return &UserController{
		authenticator: authenticator,
		userStorage:   userStorage,
		HTMLRenderer:  r,
	}
}

// @SSR
func (s *UserController) LoginPage(w http.ResponseWriter, r *http.Request) {
	s.Render(w, r, http.StatusOK, "login.tmpl.html", "loginbase", loginPageData{})
}

type loginPageData struct {
	Form loginForm
}

type loginForm struct {
	Login    string
	Password string

	validator.Validator
}

// @HTMX
func (s *UserController) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.Error(w, r, http.StatusOK, "Невалидные данные для аутентификации", nil)
		return
	}

	form := loginForm{
		Login:    r.PostForm.Get("login"),
		Password: r.PostForm.Get("password"),
	}

	form.CheckField(validator.NotBlank(form.Login), "login", "Логин не может быть пустым")
	form.CheckField(validator.NotBlank(form.Password), "password", "Пароль не может быть пустым")
	if !form.Valid() {
		data := loginPageData{Form: form}
		s.Render(w, r, http.StatusOK, "login.tmpl.html", renderer.ContentBlock, data)
		return
	}

	session, err := s.authenticator.CreateSession(r.Context(), form.Login, form.Password)
	if err != nil {
		if errors.Is(err, auth.ErrLoginPassword) {
			s.Error(w, r, http.StatusOK, "Неверный логин или пароль", nil)
			return
		}
		if errors.Is(err, auth.ErrForbidden) {
			s.Error(w, r, http.StatusOK, "Пользователю запрещен вход", nil)
			return
		}
		s.Error(w, r, http.StatusOK, "Сервис аутентификации недоступен", nil)
		return
	}

	cookie := s.authenticator.MakeSessionCookie(session.UUID)
	http.SetCookie(w, cookie)

	w.Header().Set("HX-Redirect", "/")
}

// @HTMX
func (s *UserController) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := s.authenticator.DeletionSessionCookie()
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusMovedPermanently)
}
