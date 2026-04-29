package gokalshi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClientConfig creates a ClientConfig pointing at the given test server.
func testClientConfig(t *testing.T, serverURL string) *ClientConfig {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	pemPath := filepath.Join(t.TempDir(), "test_key.pem")
	f, err := os.Create(pemPath)
	require.NoError(t, err)
	err = pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der})
	require.NoError(t, err)
	f.Close()

	creds, err := LoadCredentials("test-key", pemPath)
	require.NoError(t, err)

	return &ClientConfig{
		Environment: Demo,
		Credentials: creds,
		HTTPBaseURL: serverURL,
	}
}

// newTestClient creates a Client with a fast rate limiter and no retry delay.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cfg := testClientConfig(t, serverURL)
	limiter := NewReadWriteTokenBucket(TokenBucketConfig{
		ReadRate:  100, WriteRate: 100,
		WindowSize: 1.0, SafetyPadding: 0,
	})
	return NewClient(cfg, WithRateLimiter(limiter), WithBaseDelay(1*time.Millisecond))
}

func TestClient_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"open"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.get(context.Background(), "/trade-api/v2/exchange/status", nil)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "open")
}

func TestClient_Post_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"order_id":"abc123"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.post(context.Background(), "/trade-api/v2/portfolio/orders",
		map[string]any{"ticker": "TEST"}, 1.0)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "abc123")
}

func TestClient_AuthHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("KALSHI-ACCESS-KEY"))
		assert.NotEmpty(t, r.Header.Get("KALSHI-ACCESS-SIGNATURE"))
		assert.NotEmpty(t, r.Header.Get("KALSHI-ACCESS-TIMESTAMP"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.get(context.Background(), "/test", nil)
	require.NoError(t, err)
}

func TestClient_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "open", r.URL.Query().Get("status"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.get(context.Background(), "/test", map[string]string{
		"status": "open",
		"limit":  "10",
	})
	require.NoError(t, err)
}

func TestClient_Retry429(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"error":"rate limited"}`)
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.get(context.Background(), "/test", nil)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "ok")
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClient_Retry429_MaxExceeded(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"rate limited"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.maxRetries = 3

	_, err := c.get(context.Background(), "/test", nil)
	assert.Error(t, err)

	var rateLimitErr *RateLimitError
	assert.ErrorAs(t, err, &rateLimitErr)
	assert.Equal(t, 3, rateLimitErr.Retries)
	// Initial attempt + 3 retries = 4 total
	assert.Equal(t, int32(4), attempts.Load())
}

func TestClient_4xxError(t *testing.T) {
	// Real Kalshi API nests errors: {"error":{"code":"...","message":"..."}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"code":"bad_request","message":"invalid ticker"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.get(context.Background(), "/test", nil)
	assert.Error(t, err)

	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Equal(t, "bad_request", apiErr.Code)
	assert.Equal(t, "invalid ticker", apiErr.Message)
}

func TestClient_NetworkError(t *testing.T) {
	// Create and immediately close server to simulate connection refused
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.get(context.Background(), "/test", nil)
	assert.Error(t, err)
}

func TestClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.get(ctx, "/test", nil)
	assert.Error(t, err)
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/order123"))
		fmt.Fprint(w, `{"cancelled":true}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.delete(context.Background(), "/orders/order123", nil, 0.2)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "cancelled")
}

func TestClient_EmptyQueryParamsSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("empty"))
		assert.Equal(t, "value", r.URL.Query().Get("present"))
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.get(context.Background(), "/test", map[string]string{
		"empty":   "",
		"present": "value",
	})
	require.NoError(t, err)
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	cfg := testClientConfig(t, "http://localhost")
	c := NewClient(cfg, WithHTTPClient(custom))
	assert.Equal(t, custom, c.httpClient)
}

func TestWithMaxRetries(t *testing.T) {
	cfg := testClientConfig(t, "http://localhost")
	c := NewClient(cfg, WithMaxRetries(10))
	assert.Equal(t, 10, c.maxRetries)
}

func TestClient_Close(t *testing.T) {
	cfg := testClientConfig(t, "http://localhost")
	c := NewClient(cfg)
	c.Close() // Should not panic
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 400, Method: "GET", Path: "/test", Body: "bad request"}
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "GET")
	assert.Contains(t, err.Error(), "/test")
	assert.Contains(t, err.Error(), "bad request")
}

func TestAPIError_WithNestedError(t *testing.T) {
	err := newAPIError(400, "GET", "/test", `{"error":{"code":"invalid","message":"bad ticker"}}`)
	assert.Equal(t, "invalid", err.Code)
	assert.Equal(t, "bad ticker", err.Message)
	assert.Contains(t, err.Error(), "bad ticker")
	assert.Contains(t, err.Error(), "invalid")
}

func TestAPIError_WithFlatError(t *testing.T) {
	err := newAPIError(400, "GET", "/test", `{"code":"flat_code","message":"flat msg"}`)
	assert.Equal(t, "flat_code", err.Code)
	assert.Equal(t, "flat msg", err.Message)
}
