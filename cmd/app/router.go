package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-pkgz/routegroup"

	"github.com/agalitsyn/goth/cmd/app/templates"
	"github.com/agalitsyn/goth/pkg/httptools"
	"github.com/agalitsyn/goth/pkg/version"
)

//go:embed assets
var assets embed.FS

func MakeRouter() (*routegroup.Bundle, error) {
	router := routegroup.New(http.NewServeMux())

	// TODO: add rate limiter
	// TODO: add CSRF middleware
	router.Use(
		httptools.RequestLogger([]string{"/static", "/favicon.ico", "/robots.txt"}),
		httptools.RealIP,
		httptools.Recoverer(),
		httptools.Trace,
		httptools.AppInfo("app", version.String()),
	)

	// Note: order is important
	router.Handle("GET /static/*", fileServerHandlerFunc())
	router.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("User-agent: *\nDisallow: /"))
	})

	router.Group().Route(func(subgroup *routegroup.Bundle) {

		subgroup.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			templ.Handler(templates.IndexPage("Go + templates + HTMX", version.String(), "Petya")).ServeHTTP(w, r)
		})
	})

	return router, nil
}

func fileServerHandlerFunc() http.HandlerFunc {
	staticFS, err := fs.Sub(assets, "assets/static") // error is always nil
	if err != nil {
		panic(err) // should never happen we load from embedded FS
	}
	return func(w http.ResponseWriter, r *http.Request) {
		webFS := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
		webFS.ServeHTTP(w, r)
	}
}
