package gokalshi

import (
	"context"
	"log/slog"
)

// discardHandler is a slog.Handler that discards all log records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }

// newDiscardLogger returns a *slog.Logger that discards all output.
func newDiscardLogger() *slog.Logger {
	return slog.New(discardHandler{})
}

// WithWSLogger sets a structured logger for WebSocket lifecycle events.
func WithWSLogger(l *slog.Logger) WSClientOption {
	return func(c *WSClient) { c.logger = l }
}
