package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-pkgz/routegroup"

	"github.com/agalitsyn/goth/cmd/admin/controller"
	"github.com/agalitsyn/goth/cmd/admin/renderer"
	"github.com/agalitsyn/goth/internal/auth"
	"github.com/agalitsyn/goth/internal/model"
	"github.com/agalitsyn/goth/pkg/httptools"
	"github.com/agalitsyn/goth/pkg/version"
)

func NewRouter(
	corsMiddleware func(http.Handler) http.Handler,
	authMiddleware func(http.Handler) http.Handler,
	htmlRenderer *renderer.HTMLRenderer,
	userCtrl *controller.UserController,
) (*routegroup.Bundle, error) {
	router := routegroup.New(http.NewServeMux())

	// TODO: add rate limiter
	// TODO: add CSRF middleware
	router.Use(
		httptools.RequestLogger([]string{"/static", "/favicon.ico", "/robots.txt"}),
		httptools.RealIP,
		httptools.Recoverer(),
		corsMiddleware,
		httptools.Trace,
		httptools.AppInfo("admin", version.String()),
	)

	// Note: order is important
	router.Handle("GET /static/*", httptools.FileServerHandlerFunc(EmbedFiles, "static"))
	router.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("User-agent: *\nDisallow: /"))
	})

	router.HandleFunc("GET /login", userCtrl.LoginPage)
	router.HandleFunc("POST /login", userCtrl.Login)
	router.HandleFunc("GET /logout", userCtrl.Logout)

	router.Group().Route(func(protected *routegroup.Bundle) {
		protected.Use(authMiddleware)

		protected.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) {
			htmlRenderer.Render(w, r, http.StatusOK, "home.tmpl.html", "", nil)
		})
	})

	// Stub browser requests on favicon
	router.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	// Authenticated users will be redirected to user's type homepage or to 404 page
	// Anonymous users will be redirected to login page by middleware
	router.With(authMiddleware).HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// TODO: place for additional user checks
			_ = auth.MustUserFromContext(r.Context())
			//if user...
			htmlRenderer.Render(w, r, http.StatusOK, "home.tmpl.html", "", nil)
			return
		}

		htmlRenderer.Error(w, r, http.StatusNotFound, "Not found", nil)
	})

	return router, nil
}

func templateFuncs() template.FuncMap {
	funcs := sprig.FuncMap()
	funcs["printVersion"] = printVersion
	funcs["matchURL"] = matchURL
	return funcs
}

func printVersion() string {
	return version.String()
}

func matchURL(path string, matchTo string) bool {
	return strings.HasPrefix(path, matchTo)
}

func checkUserIsActive(user *model.User) error {
	if !user.IsActive {
		return fmt.Errorf("inactive user")
	}
	return nil
}
