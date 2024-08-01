package httptools

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// responseWriter is an http.ResponseWriter that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter creates a new responseWriter.
func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	// Default the status code to 200
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			rw := NewResponseWriter(w)
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
						slog.Info("stack", string(debug.Stack()))
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
