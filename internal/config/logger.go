package config

import (
	"fmt"
	"io"
	"log/slog"
)

// NewLogger builds a JSON slog.Logger that writes to w at the given level.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	}))
}

// replaceAttr renders duration attributes as milliseconds.
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindDuration {
		return slog.String(a.Key, fmt.Sprintf("%dms", a.Value.Duration().Milliseconds()))
	}
	return a
}
