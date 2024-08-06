package httptools

import "log/slog"

type CorsLogger struct{}

func (l CorsLogger) Printf(msg string, args ...interface{}) {
	slog.Debug("cors: "+msg, args...)
}
