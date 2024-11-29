package slogutils

import (
	"context"
	"log/slog"
	"slices"
)

type DiscardHandler struct {
	Disabled bool
	Attrs    []slog.Attr
}

func (h DiscardHandler) Enabled(context.Context, slog.Level) bool { return !h.Disabled }

func (h DiscardHandler) Handle(context.Context, slog.Record) error { return nil }

func (h DiscardHandler) WithAttrs(as []slog.Attr) slog.Handler {
	h.Attrs = slices.Concat(h.Attrs, as)
	return h
}

func (h DiscardHandler) WithGroup(name string) slog.Handler {
	return h
}
