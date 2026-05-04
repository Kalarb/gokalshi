package gokalshi

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscardLogger_NoOutput(t *testing.T) {
	logger := newDiscardLogger()

	// Should not panic and should produce no output.
	logger.Info("test message")
	logger.Error("error message")
	logger.Warn("warn message")
}

func TestDiscardHandler_Enabled(t *testing.T) {
	h := discardHandler{}
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, h.Enabled(context.Background(), slog.LevelError))
	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
}

func TestDiscardHandler_WithAttrs(t *testing.T) {
	h := discardHandler{}
	h2 := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
	assert.IsType(t, discardHandler{}, h2)
}

func TestDiscardHandler_WithGroup(t *testing.T) {
	h := discardHandler{}
	h2 := h.WithGroup("group")
	assert.IsType(t, discardHandler{}, h2)
}

func TestWithWSLogger_SetsLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := &ClientConfig{
		WSBaseURL:   "ws://localhost",
		Credentials: &Credentials{KeyID: "test"},
	}
	ws := NewWSClient(cfg, WithWSLogger(logger))

	// Verify the logger was set by triggering a log via handleIncoming with bad JSON.
	ws.handleIncoming([]byte("bad json"))

	assert.Contains(t, buf.String(), "ws_invalid_json")
}

func TestDefaultWSClient_UsesDiscardLogger(t *testing.T) {
	cfg := &ClientConfig{
		WSBaseURL:   "ws://localhost",
		Credentials: &Credentials{KeyID: "test"},
	}
	ws := NewWSClient(cfg)

	// Should not panic — discard logger silently drops.
	ws.handleIncoming([]byte("bad json"))
}
