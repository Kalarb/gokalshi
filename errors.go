package gokalshi

import (
	"encoding/json"
	"fmt"
)

// APIError represents a non-429 HTTP error from the Kalshi API.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
	Code       string // parsed from JSON response body, if available
	Message    string // parsed from JSON response body, if available
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("kalshi API error %d: %s %s: %s (code=%s)", e.StatusCode, e.Method, e.Path, e.Message, e.Code)
	}
	return fmt.Sprintf("kalshi API error %d: %s %s: %s", e.StatusCode, e.Method, e.Path, e.Body)
}

// newAPIError creates an APIError, attempting to parse code/message from the JSON body.
func newAPIError(statusCode int, method, path, body string) *APIError {
	e := &APIError{
		StatusCode: statusCode,
		Method:     method,
		Path:       path,
		Body:       body,
	}
	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		e.Code = parsed.Code
		e.Message = parsed.Message
	}
	return e
}

// RateLimitError is returned when 429 retries are exhausted.
type RateLimitError struct {
	Method  string
	Path    string
	Retries int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("kalshi rate limited: %s %s after %d retries", e.Method, e.Path, e.Retries)
}

// AuthError wraps RSA key loading and signing failures.
type AuthError struct {
	Op  string // e.g. "load_credentials", "sign", "request_headers"
	Err error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("kalshi auth error (%s): %v", e.Op, e.Err)
}

func (e *AuthError) Unwrap() error { return e.Err }

// WebSocketError wraps WS connection and protocol errors.
type WebSocketError struct {
	Op  string // e.g. "dial", "write", "read"
	Err error
}

func (e *WebSocketError) Error() string {
	return fmt.Sprintf("kalshi websocket error (%s): %v", e.Op, e.Err)
}

func (e *WebSocketError) Unwrap() error { return e.Err }

// SequenceGapError indicates a WebSocket sequence gap was detected.
type SequenceGapError struct {
	Channel  string
	Expected int
	Got      int
}

func (e *SequenceGapError) Error() string {
	return fmt.Sprintf("kalshi sequence gap on %s: expected %d, got %d", e.Channel, e.Expected, e.Got)
}
