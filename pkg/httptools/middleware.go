package httptools

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// responseWriter is an http.ResponseWriter that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger is a middleware that add http access logs
func RequestLogger(ignoredPaths []string) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			routePath := r.URL.EscapedPath()
			if routePath == "" {
				routePath = "/"
			}

			for _, p := range ignoredPaths {
				if p == routePath || strings.HasPrefix(routePath, p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			startTime := time.Now()

			rw := &responseWriter{w, http.StatusOK}
			next.ServeHTTP(rw, r)

			endTime := time.Now()

			slog.Info("http request",
				"status_code", rw.statusCode,
				"http_method", r.Method,
				"uri", r.URL.String(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"duration", endTime.Sub(startTime),
			)
		}
		return http.HandlerFunc(fn)
	}
}

// Recoverer is a middleware that recovers from panic and log it
func Recoverer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					slog.Info("request panic",
						"remote_addr", r.RemoteAddr,
						"user_agent", r.UserAgent(),
						"error", rvr)
					if rvr != http.ErrAbortHandler {
						slog.Info("request panic", "stack", string(debug.Stack()))
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// StripSlashes is a middleware that will match request paths with a trailing
// slash, strip it from the path and continue routing through the mux, if a route
// matches, then it will serve the handler.
func StripSlashes(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// TODO: remove
		fmt.Printf("(path): %#v\n", path)

		if len(path) > 1 && path[len(path)-1] == '/' {
			newPath := path[:len(path)-1]

			// TODO: remove
			fmt.Printf("(newPath): %#v\n", newPath)

			r.URL.Path = newPath
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

var (
	trueClientIP  = http.CanonicalHeaderKey("True-Client-IP")
	xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
	xRealIP       = http.CanonicalHeaderKey("X-Real-IP")
)

// RealIP is a middleware that sets a http.Request's RemoteAddr to the results
// of parsing either the True-Client-IP, X-Real-IP or the X-Forwarded-For headers
// (in that order).
//
// This middleware should be inserted fairly early in the middleware stack to
// ensure that subsequent layers (e.g., request loggers) which examine the
// RemoteAddr will see the intended value.
//
// You should only use this middleware if you can trust the headers passed to
// you (in particular, the two headers this middleware uses), for example
// because you have placed a reverse proxy like HAProxy or nginx in front of
// chi. If your reverse proxies are configured to pass along arbitrary header
// values from the client, or if you use this middleware without a reverse
// proxy, malicious clients will be able to make you very sad (or, depending on
// how you're using RemoteAddr, vulnerable to an attack of some sort).
func RealIP(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rip := realIP(r); rip != "" {
			r.RemoteAddr = rip
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func realIP(r *http.Request) string {
	var ip string

	if tcip := r.Header.Get(trueClientIP); tcip != "" {
		ip = tcip
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		ip = xrip
	} else if xff := r.Header.Get(xForwardedFor); xff != "" {
		i := strings.Index(xff, ",")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	}
	if ip == "" || net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// AppInfo adds custom app-info to the response header
func AppInfo(app, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("App-Name", app)
			w.Header().Set("App-Version", version)
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate CSRF token and store it in a cookie
		token := generateCSRFToken()
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf",
			Value:    token,
			HttpOnly: true,
			MaxAge:   24 * int(time.Hour),
			SameSite: http.SameSiteLaxMode,
		})

		// Set CSRF token in the form
		r.Header.Set("X-CSRF-Token", token)

		next.ServeHTTP(w, r)
	})
}
