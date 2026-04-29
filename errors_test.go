package gokalshi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	err := &APIError{StatusCode: 400, Method: "GET", Path: "/test", Body: "bad request"}
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "GET")
	assert.Contains(t, err.Error(), "/test")
}

func TestAPIError_WithNestedJSON(t *testing.T) {
	err := newAPIError(400, "POST", "/orders", `{"error":{"code":"invalid_ticker","message":"Ticker not found"}}`)
	assert.Equal(t, "invalid_ticker", err.Code)
	assert.Equal(t, "Ticker not found", err.Message)
	assert.Contains(t, err.Error(), "Ticker not found")
	assert.Contains(t, err.Error(), "invalid_ticker")
}

func TestAPIError_WithFlatJSON(t *testing.T) {
	err := newAPIError(400, "POST", "/orders", `{"code":"invalid_ticker","message":"Ticker not found"}`)
	assert.Equal(t, "invalid_ticker", err.Code)
	assert.Equal(t, "Ticker not found", err.Message)
}

func TestAPIError_WithInvalidJSON(t *testing.T) {
	err := newAPIError(500, "GET", "/status", "not json")
	assert.Empty(t, err.Code)
	assert.Empty(t, err.Message)
	assert.Contains(t, err.Error(), "not json")
}

func TestAPIError_ErrorAs(t *testing.T) {
	var err error = &APIError{StatusCode: 403, Method: "GET", Path: "/test"}

	var apiErr *APIError
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestRateLimitError_ErrorMessage(t *testing.T) {
	err := &RateLimitError{Method: "GET", Path: "/markets", Retries: 4}
	assert.Contains(t, err.Error(), "rate limited")
	assert.Contains(t, err.Error(), "GET")
	assert.Contains(t, err.Error(), "/markets")
	assert.Contains(t, err.Error(), "4")
}

func TestRateLimitError_ErrorAs(t *testing.T) {
	var err error = &RateLimitError{Method: "POST", Path: "/orders", Retries: 3}

	var rateLimitErr *RateLimitError
	assert.True(t, errors.As(err, &rateLimitErr))
	assert.Equal(t, 3, rateLimitErr.Retries)
}

func TestAuthError_ErrorMessage(t *testing.T) {
	err := &AuthError{Op: "load_credentials", Err: fmt.Errorf("file not found")}
	assert.Contains(t, err.Error(), "auth error")
	assert.Contains(t, err.Error(), "load_credentials")
	assert.Contains(t, err.Error(), "file not found")
}

func TestAuthError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &AuthError{Op: "sign", Err: inner}

	assert.True(t, errors.Is(err, inner))
}

func TestAuthError_ErrorAs(t *testing.T) {
	var err error = &AuthError{Op: "sign", Err: fmt.Errorf("key error")}

	var authErr *AuthError
	assert.True(t, errors.As(err, &authErr))
	assert.Equal(t, "sign", authErr.Op)
}

func TestWebSocketError_ErrorMessage(t *testing.T) {
	err := &WebSocketError{Op: "dial", Err: fmt.Errorf("connection refused")}
	assert.Contains(t, err.Error(), "websocket error")
	assert.Contains(t, err.Error(), "dial")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestWebSocketError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner ws error")
	err := &WebSocketError{Op: "write", Err: inner}

	assert.True(t, errors.Is(err, inner))
}

func TestWebSocketError_ErrorAs(t *testing.T) {
	var err error = &WebSocketError{Op: "read", Err: fmt.Errorf("timeout")}

	var wsErr *WebSocketError
	assert.True(t, errors.As(err, &wsErr))
	assert.Equal(t, "read", wsErr.Op)
}

func TestSequenceGapError_ErrorMessage(t *testing.T) {
	err := &SequenceGapError{Channel: "orderbook_delta", Expected: 5, Got: 10}
	assert.Contains(t, err.Error(), "sequence gap")
	assert.Contains(t, err.Error(), "orderbook_delta")
	assert.Contains(t, err.Error(), "5")
	assert.Contains(t, err.Error(), "10")
}

func TestSequenceGapError_ErrorAs(t *testing.T) {
	var err error = &SequenceGapError{Channel: "ticker", Expected: 1, Got: 5}

	var seqErr *SequenceGapError
	assert.True(t, errors.As(err, &seqErr))
	assert.Equal(t, "ticker", seqErr.Channel)
	assert.Equal(t, 1, seqErr.Expected)
	assert.Equal(t, 5, seqErr.Got)
}
