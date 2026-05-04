package gokalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is the Kalshi HTTP API client.
// It wraps net/http with authentication, rate limiting, and retry logic.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	credentials *Credentials
	limiter     *ReadWriteTokenBucket
	maxRetries  int
	baseDelay   time.Duration
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cl *Client) { cl.httpClient = c }
}

// WithMaxRetries sets the max number of 429 retries.
func WithMaxRetries(n int) ClientOption {
	return func(cl *Client) { cl.maxRetries = n }
}

// WithBaseDelay sets the base retry delay for 429 responses.
func WithBaseDelay(d time.Duration) ClientOption {
	return func(cl *Client) { cl.baseDelay = d }
}

// WithRateLimiter sets a custom rate limiter.
func WithRateLimiter(l *ReadWriteTokenBucket) ClientOption {
	return func(cl *Client) { cl.limiter = l }
}

// NewClient creates a new Kalshi HTTP client from a ClientConfig.
func NewClient(cfg *ClientConfig, opts ...ClientOption) *Client {
	c := &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     cfg.HTTPBaseURL,
		credentials: cfg.Credentials,
		limiter:     mustNewTokenBucket(DefaultTokenBucketConfig()),
		maxRetries:  4,
		baseDelay:   100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Close releases resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

func mustNewTokenBucket(cfg TokenBucketConfig) *ReadWriteTokenBucket {
	b, err := NewReadWriteTokenBucket(cfg)
	if err != nil {
		panic("invalid default token bucket config: " + err.Error())
	}
	return b
}

// do executes an HTTP request with rate limiting, auth headers, and 429 retry.
func (c *Client) do(ctx context.Context, method, path string, readCost, writeCost float64, body any, params map[string]string) (json.RawMessage, error) {
	if err := c.limiter.Acquire(ctx, readCost, writeCost); err != nil {
		return nil, fmt.Errorf("rate limit acquire: %w", err)
	}

	fullURL := buildURL(c.baseURL, path, params)

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	for retries := 0; ; retries++ {
		req, err := c.buildAuthRequest(ctx, method, path, fullURL, bodyBytes)
		if err != nil {
			return nil, err
		}

		respBody, statusCode, err := c.executeRequest(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP %s %s: %w", method, path, err)
		}

		if statusCode == http.StatusTooManyRequests || isRetryableStatus(statusCode) {
			if retried, retryErr := c.handleRetry(ctx, method, path, retries); retryErr != nil {
				// Exhausted retries — return the original API error for 5xx.
				if isRetryableStatus(statusCode) {
					return nil, newAPIError(statusCode, method, path, string(respBody))
				}
				return nil, retryErr
			} else if retried {
				continue
			}
		}

		if statusCode >= 400 {
			return nil, newAPIError(statusCode, method, path, string(respBody))
		}

		return json.RawMessage(respBody), nil
	}
}

// buildURL constructs the full URL with query parameters.
func buildURL(baseURL, path string, params map[string]string) string {
	fullURL := baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			if v != "" {
				q.Set(k, v)
			}
		}
		fullURL += "?" + q.Encode()
	}
	return fullURL
}

// buildAuthRequest creates an authenticated HTTP request.
func (c *Client) buildAuthRequest(ctx context.Context, method, path, fullURL string, bodyBytes []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	headers, err := c.credentials.RequestHeaders(method, path)
	if err != nil {
		return nil, fmt.Errorf("generate auth headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// executeRequest sends the request and returns the response body and status code.
func (c *Client) executeRequest(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// isRetryableStatus returns true for server errors worth retrying.
func isRetryableStatus(code int) bool {
	return code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// handleRetry handles 429 rate limit responses with exponential backoff.
// Returns (true, nil) if a retry should be attempted, or (false, error) if exhausted/cancelled.
func (c *Client) handleRetry(ctx context.Context, method, path string, retries int) (bool, error) {
	if retries >= c.maxRetries {
		return false, &RateLimitError{Method: method, Path: path, Retries: retries}
	}

	shift := retries
	if shift > 30 {
		shift = 30
	}
	wait := c.baseDelay * (1 << shift)

	select {
	case <-ctx.Done():
		return false, fmt.Errorf("retry cancelled: %w", ctx.Err())
	case <-time.After(wait):
		return true, nil
	}
}

// doJSON executes an HTTP request and unmarshals the response into T.
func doJSON[T any](c *Client, ctx context.Context, method, path string, readCost, writeCost float64, body any, params map[string]string) (T, error) {
	raw, err := c.do(ctx, method, path, readCost, writeCost, body, params)
	var result T
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("unmarshal %s %s response: %w", method, path, err)
	}
	return result, nil
}

// Convenience methods for HTTP verbs.

func (c *Client) get(ctx context.Context, path string, params map[string]string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, path, 1.0, 0, nil, params)
}

func getJSON[T any](c *Client, ctx context.Context, path string, params map[string]string) (T, error) {
	return doJSON[T](c, ctx, http.MethodGet, path, 1.0, 0, nil, params)
}

func (c *Client) post(ctx context.Context, path string, body any, writeCost float64) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, 0, writeCost, body, nil)
}

func postJSON[T any](c *Client, ctx context.Context, path string, body any, writeCost float64) (T, error) {
	return doJSON[T](c, ctx, http.MethodPost, path, 0, writeCost, body, nil)
}

func (c *Client) delete(ctx context.Context, path string, body any, writeCost float64) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, path, 0, writeCost, body, nil)
}

func deleteJSON[T any](c *Client, ctx context.Context, path string, body any, writeCost float64) (T, error) {
	return doJSON[T](c, ctx, http.MethodDelete, path, 0, writeCost, body, nil)
}
