package slogtools

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func ParseLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case slog.LevelDebug.String():
		return slog.LevelDebug
	case slog.LevelWarn.String():
		return slog.LevelWarn
	case slog.LevelError.String():
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func SetupGlobalLogger(level slog.Level, out *os.File) {
	logger := slog.New(
		tint.NewHandler(out, &tint.Options{
			Level:      level,
			TimeFormat: time.DateTime,
			NoColor:    !isatty.IsTerminal(out.Fd()),
		}),
	)
	slog.SetDefault(logger)
}

func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
