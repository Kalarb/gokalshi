package gokalshi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

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

// newTestClient creates a Client with a fast rate limiter, no retry delay,
// and auto-config disabled (WithRateLimiter skips the startup API calls).
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cfg := testClientConfig(t, serverURL)
	limiter := NewReadWriteTokenBucket(TokenBucketConfig{
		ReadRate: 100, WriteRate: 100,
		WindowSize: 1.0, SafetyPadding: 0,
	})
	c, err := NewClient(cfg, WithRateLimiter(limiter), WithBaseDelay(1*time.Millisecond))
	require.NoError(t, err)
	return c
}

// ---------------------------------------------------------------------------
// HTTP client core tests
// ---------------------------------------------------------------------------

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
	assert.Equal(t, int32(4), attempts.Load())
}

func TestClient_4xxError(t *testing.T) {
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
	limiter := NewReadWriteTokenBucket(DefaultTokenBucketConfig())
	c, err := NewClient(cfg, WithHTTPClient(custom), WithRateLimiter(limiter))
	require.NoError(t, err)
	assert.Equal(t, custom, c.httpClient)
}

func TestWithMaxRetries(t *testing.T) {
	cfg := testClientConfig(t, "http://localhost")
	limiter := NewReadWriteTokenBucket(DefaultTokenBucketConfig())
	c, err := NewClient(cfg, WithMaxRetries(10), WithRateLimiter(limiter))
	require.NoError(t, err)
	assert.Equal(t, 10, c.maxRetries)
}

func TestClient_Close(t *testing.T) {
	cfg := testClientConfig(t, "http://localhost")
	limiter := NewReadWriteTokenBucket(DefaultTokenBucketConfig())
	c, err := NewClient(cfg, WithRateLimiter(limiter))
	require.NoError(t, err)
	c.Close()
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

// ---------------------------------------------------------------------------
// API endpoint tests
// ---------------------------------------------------------------------------

func TestGetExchangeStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/exchange/status", r.URL.Path)
		fmt.Fprint(w, `{"exchange_active":true}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetExchangeStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, resp.ExchangeActive)
}

func TestGetMarketOrderbook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/markets/KXFED/orderbook", r.URL.Path)
		fmt.Fprint(w, `{"orderbook_fp":{"yes_dollars":[],"no_dollars":[]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMarketOrderbook(context.Background(), "KXFED", GetOrderbookParams{})
	require.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "TICK", r.URL.Query().Get("ticker"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		fmt.Fprint(w, `{"trades":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetTrades(context.Background(), GetTradesParams{Ticker: "TICK", Limit: 5})
	require.NoError(t, err)
}

func TestGetTradesParams_toMap_AllFields(t *testing.T) {
	p := GetTradesParams{Ticker: "T", Limit: 10, Cursor: "c", MinTs: 100, MaxTs: 200}
	m := p.toMap()
	assert.Equal(t, "T", m["ticker"])
	assert.Equal(t, "10", m["limit"])
	assert.Equal(t, "c", m["cursor"])
	assert.Equal(t, "100", m["min_ts"])
	assert.Equal(t, "200", m["max_ts"])
}

func TestGetMarket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/markets/ABC", r.URL.Path)
		fmt.Fprint(w, `{"market":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMarket(context.Background(), "ABC")
	require.NoError(t, err)
}

func TestGetMarkets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "EVT", r.URL.Query().Get("event_ticker"))
		assert.Equal(t, "SER", r.URL.Query().Get("series_ticker"))
		assert.Equal(t, "open", r.URL.Query().Get("status"))
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		assert.Equal(t, "cur", r.URL.Query().Get("cursor"))
		fmt.Fprint(w, `{"markets":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMarkets(context.Background(), GetMarketsParams{
		Limit: 20, Cursor: "cur", EventTicker: "EVT", SeriesTicker: "SER", Status: "open",
	})
	require.NoError(t, err)
}

func TestGetMarketsParams_toMap_Empty(t *testing.T) {
	m := GetMarketsParams{}.toMap()
	assert.Empty(t, m)
}

func TestGetBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/balance", r.URL.Path)
		fmt.Fprint(w, `{"balance":1000}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetBalance(context.Background())
	require.NoError(t, err)
}

func TestGetPositions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/positions", r.URL.Path)
		assert.Equal(t, "TICK", r.URL.Query().Get("ticker"))
		fmt.Fprint(w, `{"positions":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetPositions(context.Background(), GetPositionsParams{Ticker: "TICK"})
	require.NoError(t, err)
}

func TestGetPositionsParams_toMap_AllFields(t *testing.T) {
	p := GetPositionsParams{Ticker: "T", EventTicker: "E", CountFilter: "yes", Limit: 5, Cursor: "c"}
	m := p.toMap()
	assert.Equal(t, "T", m["ticker"])
	assert.Equal(t, "E", m["event_ticker"])
	assert.Equal(t, "yes", m["count_filter"])
	assert.Equal(t, "5", m["limit"])
	assert.Equal(t, "c", m["cursor"])
}

func TestGetFills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/fills", r.URL.Path)
		fmt.Fprint(w, `{"fills":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetFills(context.Background(), GetFillsParams{})
	require.NoError(t, err)
}

func TestGetEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/events/EVT1", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("with_nested_markets"))
		fmt.Fprint(w, `{"event":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetEvent(context.Background(), "EVT1", GetEventParams{WithNestedMarkets: true})
	require.NoError(t, err)
}

func TestGetEventParams_toMap_Empty(t *testing.T) {
	m := GetEventParams{}.toMap()
	assert.Empty(t, m)
}

func TestGetEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/events", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "SER", r.URL.Query().Get("series_ticker"))
		assert.Equal(t, "open", r.URL.Query().Get("status"))
		assert.Equal(t, "true", r.URL.Query().Get("with_nested_markets"))
		assert.Equal(t, "cur", r.URL.Query().Get("cursor"))
		fmt.Fprint(w, `{"events":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetEvents(context.Background(), GetEventsParams{
		Limit: 10, Cursor: "cur", WithNestedMarkets: true, Status: "open", SeriesTicker: "SER",
	})
	require.NoError(t, err)
}

func TestGetEventsParams_toMap_Empty(t *testing.T) {
	m := GetEventsParams{}.toMap()
	assert.Empty(t, m)
}

func TestGetSeries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/series/SER1", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("include_volume"))
		fmt.Fprint(w, `{"series":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetSeries(context.Background(), "SER1", GetSeriesParams{IncludeVolume: true})
	require.NoError(t, err)
}

func TestGetSeriesParams_toMap_Empty(t *testing.T) {
	m := GetSeriesParams{}.toMap()
	assert.Empty(t, m)
}

func TestGetSeriesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/series", r.URL.Path)
		assert.Equal(t, "politics", r.URL.Query().Get("category"))
		fmt.Fprint(w, `{"series":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetSeriesList(context.Background(), GetSeriesListParams{Category: "politics"})
	require.NoError(t, err)
}

func TestGetSeriesListParams_toMap_Empty(t *testing.T) {
	m := GetSeriesListParams{}.toMap()
	assert.Empty(t, m)
}

// ---------------------------------------------------------------------------
// Order-specific tests
// ---------------------------------------------------------------------------

func TestCreateOrder_PayloadSent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, pathOrders, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "TEST-TICKER", payload["ticker"])
		assert.Equal(t, "buy", payload["action"])

		fmt.Fprint(w, `{"order":{"order_id":"ord_123"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.CreateOrder(context.Background(), CreateOrderRequest{
		Ticker: "TEST-TICKER",
		Action: "buy",
		Side:   "yes",
	})
	require.NoError(t, err)
	assert.Equal(t, "ord_123", resp.Order.OrderID)
}

func TestCancelOrder_Path(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, pathOrders+"/order-456", r.URL.Path)
		fmt.Fprint(w, `{"order":{"order_id":"order-456","status":"canceled"},"reduced_by_fp":"5.00"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.CancelOrder(context.Background(), "order-456")
	require.NoError(t, err)
}

func TestGetOrder_Path(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, pathOrders+"/order-789", r.URL.Path)
		fmt.Fprint(w, `{"order":{"order_id":"order-789"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetOrder(context.Background(), "order-789")
	require.NoError(t, err)
}

func TestGetOrders_QueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "TEST", r.URL.Query().Get("ticker"))
		assert.Equal(t, "resting", r.URL.Query().Get("status"))
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		fmt.Fprint(w, `{"orders":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetOrders(context.Background(), GetOrdersParams{
		Ticker: "TEST",
		Status: "resting",
		Limit:  50,
	})
	require.NoError(t, err)
}

func TestBatchCreateOrders_BodyFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		orders, ok := payload["orders"].([]any)
		require.True(t, ok)
		assert.Len(t, orders, 2)

		fmt.Fprint(w, `{"orders":[{},{}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.BatchCreateOrders(context.Background(), []CreateOrderRequest{
		{Ticker: "A", Action: "buy"},
		{Ticker: "B", Action: "sell"},
	})
	require.NoError(t, err)
}

func TestBatchCancelOrders_BodyFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))

		orders, ok := payload["orders"].([]any)
		require.True(t, ok)
		assert.Len(t, orders, 3)

		fmt.Fprint(w, `{"orders":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.BatchCancelOrders(context.Background(), []BatchCancelOrdersRequestOrder{
		{OrderID: "a"}, {OrderID: "b"}, {OrderID: "c"},
	})
	require.NoError(t, err)
}

func TestAmendOrder_Path(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, pathOrders+"/ord-1/amend", r.URL.Path)
		fmt.Fprint(w, `{"old_order":{"order_id":"ord-1"},"order":{"order_id":"ord-1"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.AmendOrder(context.Background(), "ord-1", AmendOrderRequest{
		Ticker: "TEST", Side: "yes", Action: "buy", CountFP: ptr("5.00"),
	})
	require.NoError(t, err)
}

func TestDecreaseOrder_Path(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, pathOrders+"/ord-2/decrease", r.URL.Path)
		fmt.Fprint(w, `{"order":{"order_id":"ord-2"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.DecreaseOrder(context.Background(), "ord-2", DecreaseOrderRequest{
		ReduceByFP: ptr("3.00"),
	})
	require.NoError(t, err)
}

func TestGetOrdersParams_toMap_AllFields(t *testing.T) {
	p := GetOrdersParams{
		Ticker: "T", EventTicker: "E", Status: "resting",
		Limit: 10, Cursor: "cur", MinTs: 100, MaxTs: 200,
	}
	m := p.toMap()
	assert.Equal(t, "T", m["ticker"])
	assert.Equal(t, "E", m["event_ticker"])
	assert.Equal(t, "resting", m["status"])
	assert.Equal(t, "10", m["limit"])
	assert.Equal(t, "cur", m["cursor"])
	assert.Equal(t, "100", m["min_ts"])
	assert.Equal(t, "200", m["max_ts"])
}

// ---------------------------------------------------------------------------
// Account endpoint tests
// ---------------------------------------------------------------------------

func TestGetAccountAPILimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/account/limits", r.URL.Path)
		fmt.Fprint(w, `{"usage_tier":"basic","read":{"refill_rate":20},"write":{"refill_rate":10}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetAccountAPILimits(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "basic", resp.UsageTier)
}

// ---------------------------------------------------------------------------
// Exchange endpoint tests (missing 4)
// ---------------------------------------------------------------------------

func TestGetExchangeAnnouncements(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/exchange/announcements", r.URL.Path)
		fmt.Fprint(w, `{"announcements":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetExchangeAnnouncements(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp.Announcements)
}

func TestGetExchangeSchedule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/exchange/schedule", r.URL.Path)
		fmt.Fprint(w, `{"schedule":{"standard_hours":[{"monday":[{"open_time":"09:00","close_time":"17:00"}]}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetExchangeSchedule(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Schedule.StandardHours)
}

func TestGetUserDataTimestamp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/exchange/user_data_timestamp", r.URL.Path)
		fmt.Fprint(w, `{"as_of_time":"2024-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetUserDataTimestamp(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", resp.AsOfTime)
}

func TestGetSeriesFeeChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/series/fee_changes", r.URL.Path)
		assert.Equal(t, "KXBTC", r.URL.Query().Get("series_ticker"))
		fmt.Fprint(w, `{"series_fee_change_arr":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetSeriesFeeChanges(context.Background(), GetSeriesFeeChangesParams{
		SeriesTicker: "KXBTC",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp.SeriesFeeChangeArr)
}

func TestGetSeriesFeeChangesParams_toMap(t *testing.T) {
	p := GetSeriesFeeChangesParams{SeriesTicker: "S", ShowHistorical: true}
	m := p.toMap()
	assert.Equal(t, "S", m["series_ticker"])
	assert.Equal(t, "true", m["show_historical"])
}

// ---------------------------------------------------------------------------
// Orders endpoint tests (missing 2)
// ---------------------------------------------------------------------------

func TestGetQueuePositions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/orders/queue_positions", r.URL.Path)
		assert.Equal(t, "TICK-A", r.URL.Query().Get("market_tickers"))
		fmt.Fprint(w, `{"queue_positions":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetQueuePositions(context.Background(), GetQueuePositionsParams{
		MarketTickers: "TICK-A",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp.QueuePositions)
}

func TestGetQueuePosition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, pathOrders+"/ord-99/queue_position", r.URL.Path)
		fmt.Fprint(w, `{"queue_position_fp":"42.00"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetQueuePosition(context.Background(), "ord-99")
	require.NoError(t, err)
	assert.Equal(t, "42.00", resp.QueuePositionFP)
}

func TestGetQueuePositionsParams_toMap(t *testing.T) {
	p := GetQueuePositionsParams{MarketTickers: "A,B", EventTicker: "E"}
	m := p.toMap()
	assert.Equal(t, "A,B", m["market_tickers"])
	assert.Equal(t, "E", m["event_ticker"])
}

// ---------------------------------------------------------------------------
// Portfolio endpoint tests (missing 1)
// ---------------------------------------------------------------------------

func TestGetSettlements(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/settlements", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		fmt.Fprint(w, `{"settlements":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetSettlements(context.Background(), GetSettlementsParams{Limit: 10})
	require.NoError(t, err)
	assert.NotNil(t, resp.Settlements)
}

func TestGetSettlementsParams_toMap(t *testing.T) {
	p := GetSettlementsParams{Ticker: "T", EventTicker: "E", Limit: 5, Cursor: "c", MinTs: 1, MaxTs: 2}
	m := p.toMap()
	assert.Equal(t, "T", m["ticker"])
	assert.Equal(t, "E", m["event_ticker"])
	assert.Equal(t, "5", m["limit"])
	assert.Equal(t, "c", m["cursor"])
	assert.Equal(t, "1", m["min_ts"])
	assert.Equal(t, "2", m["max_ts"])
}

// ---------------------------------------------------------------------------
// Markets endpoint tests (missing 3)
// ---------------------------------------------------------------------------

func TestGetMarketOrderbooks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/markets/orderbooks", r.URL.Path)
		assert.Equal(t, "A,B,C", r.URL.Query().Get("tickers"))
		fmt.Fprint(w, `{"orderbooks":[{},{}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetMarketOrderbooks(context.Background(), GetMarketOrderbooksParams{
		Tickers: "A,B,C",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Orderbooks, 2)
}

func TestGetMarketCandlesticks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/series/SER/markets/MKT/candlesticks", r.URL.Path)
		assert.Equal(t, "1000", r.URL.Query().Get("start_ts"))
		assert.Equal(t, "2000", r.URL.Query().Get("end_ts"))
		assert.Equal(t, "60", r.URL.Query().Get("period_interval"))
		fmt.Fprint(w, `{"ticker":"MKT","candlesticks":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetMarketCandlesticks(context.Background(), "SER", "MKT", GetMarketCandlesticksParams{
		StartTs: 1000, EndTs: 2000, PeriodInterval: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, "MKT", resp.Ticker)
}

func TestGetBatchMarketCandlesticks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/markets/candlesticks", r.URL.Path)
		assert.Equal(t, "A,B", r.URL.Query().Get("market_tickers"))
		assert.Equal(t, "1", r.URL.Query().Get("period_interval"))
		fmt.Fprint(w, `{"markets":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetBatchMarketCandlesticks(context.Background(), GetBatchMarketCandlesticksParams{
		MarketTickers: "A,B", StartTs: 100, EndTs: 200, PeriodInterval: 1,
	})
	require.NoError(t, err)
}

func TestGetMarketOrderbooksParams_toMap(t *testing.T) {
	p := GetMarketOrderbooksParams{Tickers: "X,Y"}
	m := p.toMap()
	assert.Equal(t, "X,Y", m["tickers"])
}

func TestGetMarketCandlesticksParams_toMap(t *testing.T) {
	p := GetMarketCandlesticksParams{StartTs: 1, EndTs: 2, PeriodInterval: 60, IncludeLatestBeforeStart: true}
	m := p.toMap()
	assert.Equal(t, "1", m["start_ts"])
	assert.Equal(t, "2", m["end_ts"])
	assert.Equal(t, "60", m["period_interval"])
	assert.Equal(t, "true", m["include_latest_before_start"])
}

func TestGetBatchMarketCandlesticksParams_toMap(t *testing.T) {
	p := GetBatchMarketCandlesticksParams{MarketTickers: "A,B", StartTs: 1, EndTs: 2, PeriodInterval: 1440}
	m := p.toMap()
	assert.Equal(t, "A,B", m["market_tickers"])
	assert.Equal(t, "1", m["start_ts"])
	assert.Equal(t, "2", m["end_ts"])
	assert.Equal(t, "1440", m["period_interval"])
}

// ---------------------------------------------------------------------------
// Events endpoint tests (missing 4)
// ---------------------------------------------------------------------------

func TestGetEventMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/events/EVT-1/metadata", r.URL.Path)
		fmt.Fprint(w, `{"image_url":"https://example.com/img.png"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetEventMetadata(context.Background(), "EVT-1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/img.png", resp.ImageURL)
}

func TestGetMultivariateEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/events/multivariate", r.URL.Path)
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "SER", r.URL.Query().Get("series_ticker"))
		fmt.Fprint(w, `{"events":[],"cursor":""}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetMultivariateEvents(context.Background(), GetMultivariateEventsParams{
		Limit: 5, SeriesTicker: "SER",
	})
	require.NoError(t, err)
	assert.NotNil(t, resp.Events)
}

func TestGetEventCandlesticks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/series/SER/events/EVT/candlesticks", r.URL.Path)
		assert.Equal(t, "1000", r.URL.Query().Get("start_ts"))
		assert.Equal(t, "2000", r.URL.Query().Get("end_ts"))
		assert.Equal(t, "1", r.URL.Query().Get("period_interval"))
		fmt.Fprint(w, `{"market_candlesticks":[],"market_tickers":[],"adjusted_end_ts":0}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetEventCandlesticks(context.Background(), "SER", "EVT", GetEventCandlesticksParams{
		StartTs: 1000, EndTs: 2000, PeriodInterval: 1,
	})
	require.NoError(t, err)
}

func TestGetEventForecastPercentileHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/series/SER/events/EVT/forecast_percentile_history", r.URL.Path)
		assert.Equal(t, "2500,5000,7500", r.URL.Query().Get("percentiles"))
		assert.Equal(t, "60", r.URL.Query().Get("period_interval"))
		fmt.Fprint(w, `{"forecast_history":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetEventForecastPercentileHistory(context.Background(), "SER", "EVT", GetEventForecastPercentileHistoryParams{
		Percentiles: "2500,5000,7500", StartTs: 100, EndTs: 200, PeriodInterval: 60,
	})
	require.NoError(t, err)
	assert.NotNil(t, resp.ForecastHistory)
}

func TestGetMultivariateEventsParams_toMap(t *testing.T) {
	p := GetMultivariateEventsParams{Limit: 5, Cursor: "c", SeriesTicker: "S", CollectionTicker: "C", WithNestedMarkets: true}
	m := p.toMap()
	assert.Equal(t, "5", m["limit"])
	assert.Equal(t, "c", m["cursor"])
	assert.Equal(t, "S", m["series_ticker"])
	assert.Equal(t, "C", m["collection_ticker"])
	assert.Equal(t, "true", m["with_nested_markets"])
}

func TestGetEventCandlesticksParams_toMap(t *testing.T) {
	p := GetEventCandlesticksParams{StartTs: 1, EndTs: 2, PeriodInterval: 60}
	m := p.toMap()
	assert.Equal(t, "1", m["start_ts"])
	assert.Equal(t, "2", m["end_ts"])
	assert.Equal(t, "60", m["period_interval"])
}

func TestGetEventForecastPercentileHistoryParams_toMap(t *testing.T) {
	p := GetEventForecastPercentileHistoryParams{Percentiles: "25,50", StartTs: 1, EndTs: 2, PeriodInterval: 1}
	m := p.toMap()
	assert.Equal(t, "25,50", m["percentiles"])
	assert.Equal(t, "1", m["start_ts"])
	assert.Equal(t, "2", m["end_ts"])
	assert.Equal(t, "1", m["period_interval"])
}

// ---------------------------------------------------------------------------
// Search endpoint tests (missing 2)
// ---------------------------------------------------------------------------

func TestGetTagsByCategories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/search/tags_by_categories", r.URL.Path)
		fmt.Fprint(w, `{"tags_by_categories":{"Politics":["tag1"]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetTagsByCategories(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, resp.TagsByCategories)
}

func TestGetFiltersBySport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/trade-api/v2/search/filters_by_sport", r.URL.Path)
		fmt.Fprint(w, `{"filters_by_sports":{},"sport_ordering":["NFL"]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetFiltersBySport(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.SportOrdering, 1)
	assert.Equal(t, "NFL", resp.SportOrdering[0])
}

// ---------------------------------------------------------------------------
// Account endpoint tests (new)
// ---------------------------------------------------------------------------

func TestGetAccountEndpointCosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/account/endpoint_costs", r.URL.Path)
		fmt.Fprint(w, `{"default_cost":1,"endpoint_costs":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetAccountEndpointCosts(context.Background())
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Portfolio Summary endpoint test
// ---------------------------------------------------------------------------

func TestGetPortfolioRestingOrderTotalValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/summary/total_resting_order_value", r.URL.Path)
		fmt.Fprint(w, `{"total_resting_order_value":500}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetPortfolioRestingOrderTotalValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 500, resp.TotalRestingOrderValue)
}

// ---------------------------------------------------------------------------
// Portfolio deposits / withdrawals tests
// ---------------------------------------------------------------------------

func TestGetDeposits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/deposits", r.URL.Path)
		fmt.Fprint(w, `{"deposits":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetDeposits(context.Background(), GetDepositsParams{Limit: 10})
	require.NoError(t, err)
}

func TestGetWithdrawals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/withdrawals", r.URL.Path)
		fmt.Fprint(w, `{"withdrawals":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetWithdrawals(context.Background(), GetWithdrawalsParams{Limit: 10})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Subaccounts endpoint tests
// ---------------------------------------------------------------------------

func TestCreateSubaccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts", r.URL.Path)
		fmt.Fprint(w, `{"subaccount_number":1}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.CreateSubaccount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, resp.SubaccountNumber)
}

func TestGetSubaccountBalances(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts/balances", r.URL.Path)
		fmt.Fprint(w, `{"subaccount_balances":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetSubaccountBalances(context.Background())
	require.NoError(t, err)
}

func TestGetSubaccountNetting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts/netting", r.URL.Path)
		fmt.Fprint(w, `{"netting_configs":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetSubaccountNetting(context.Background())
	require.NoError(t, err)
}

func TestUpdateSubaccountNetting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts/netting", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.UpdateSubaccountNetting(context.Background(), UpdateSubaccountNettingRequest{
		Enabled:          true,
		SubaccountNumber: 1,
	})
	require.NoError(t, err)
}

func TestApplySubaccountTransfer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts/transfer", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.ApplySubaccountTransfer(context.Background(), ApplySubaccountTransferRequest{
		AmountCents:    100,
		FromSubaccount: 0,
		ToSubaccount:   1,
	})
	require.NoError(t, err)
}

func TestGetSubaccountTransfers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/subaccounts/transfers", r.URL.Path)
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		fmt.Fprint(w, `{"transfers":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetSubaccountTransfers(context.Background(), GetSubaccountTransfersParams{Limit: 5})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// API Keys endpoint tests
// ---------------------------------------------------------------------------

func TestGetAPIKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/api_keys", r.URL.Path)
		fmt.Fprint(w, `{"api_keys":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetAPIKeys(context.Background())
	require.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/api_keys", r.URL.Path)
		fmt.Fprint(w, `{"api_key":"key-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.CreateAPIKey(context.Background(), CreateApiKeyRequest{})
	require.NoError(t, err)
}

func TestGenerateAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/api_keys/generate", r.URL.Path)
		fmt.Fprint(w, `{"api_key":"key-2"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GenerateAPIKey(context.Background(), GenerateApiKeyRequest{})
	require.NoError(t, err)
}

func TestDeleteAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/api_keys/key-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteAPIKey(context.Background(), "key-1")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Event Orders V2 endpoint tests
// ---------------------------------------------------------------------------

func TestCreateOrderV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders", r.URL.Path)
		fmt.Fprint(w, `{"order":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.CreateOrderV2(context.Background(), CreateOrderV2Request{})
	require.NoError(t, err)
}

func TestBatchCreateOrdersV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders/batched", r.URL.Path)
		fmt.Fprint(w, `{"orders":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.BatchCreateOrdersV2(context.Background(), BatchCreateOrdersV2Request{
		Orders: []CreateOrderV2Request{{}},
	})
	require.NoError(t, err)
}

func TestBatchCancelOrdersV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders/batched", r.URL.Path)
		fmt.Fprint(w, `{"orders":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.BatchCancelOrdersV2(context.Background(), BatchCancelOrdersV2Request{
		Orders: []map[string]any{{"order_id": "ord-1"}},
	})
	require.NoError(t, err)
}

func TestCancelOrderV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders/ord-1", r.URL.Path)
		fmt.Fprint(w, `{"order":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.CancelOrderV2(context.Background(), "ord-1", CancelOrderV2Params{})
	require.NoError(t, err)
}

func TestAmendOrderV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders/ord-1/amend", r.URL.Path)
		fmt.Fprint(w, `{"order":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.AmendOrderV2(context.Background(), "ord-1", AmendOrderV2Request{})
	require.NoError(t, err)
}

func TestDecreaseOrderV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/events/orders/ord-1/decrease", r.URL.Path)
		fmt.Fprint(w, `{"order":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.DecreaseOrderV2(context.Background(), "ord-1", DecreaseOrderV2Request{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Events fee changes test
// ---------------------------------------------------------------------------

func TestGetEventFeeChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/events/fee_changes", r.URL.Path)
		assert.Equal(t, "EVT", r.URL.Query().Get("event_ticker"))
		fmt.Fprint(w, `{"event_fee_changes":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetEventFeeChanges(context.Background(), GetEventFeeChangesParams{EventTicker: "EVT"})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Historical endpoint tests
// ---------------------------------------------------------------------------

func TestGetHistoricalCutoff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/cutoff", r.URL.Path)
		fmt.Fprint(w, `{"market_settled_ts":"2024-01-01","orders_updated_ts":"2024-01-01","trades_created_ts":"2024-01-01"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetHistoricalCutoff(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, resp.MarketSettledTS)
}

func TestGetHistoricalFills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/fills", r.URL.Path)
		fmt.Fprint(w, `{"fills":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalFills(context.Background(), GetHistoricalFillsParams{Limit: 10})
	require.NoError(t, err)
}

func TestGetHistoricalOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/orders", r.URL.Path)
		fmt.Fprint(w, `{"orders":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalOrders(context.Background(), GetHistoricalOrdersParams{})
	require.NoError(t, err)
}

func TestGetHistoricalTrades(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/trades", r.URL.Path)
		fmt.Fprint(w, `{"trades":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalTrades(context.Background(), GetHistoricalTradesParams{})
	require.NoError(t, err)
}

func TestGetHistoricalMarkets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/markets", r.URL.Path)
		fmt.Fprint(w, `{"markets":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalMarkets(context.Background(), GetHistoricalMarketsParams{})
	require.NoError(t, err)
}

func TestGetHistoricalMarket(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/markets/TICK-1", r.URL.Path)
		fmt.Fprint(w, `{"market":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalMarket(context.Background(), "TICK-1")
	require.NoError(t, err)
}

func TestGetHistoricalMarketCandlesticks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/historical/markets/TICK-1/candlesticks", r.URL.Path)
		fmt.Fprint(w, `{"candlesticks":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetHistoricalMarketCandlesticks(context.Background(), "TICK-1", GetHistoricalMarketCandlesticksParams{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Incentive Programs endpoint test
// ---------------------------------------------------------------------------

func TestGetIncentivePrograms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/incentive_programs", r.URL.Path)
		fmt.Fprint(w, `{"incentive_programs":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetIncentivePrograms(context.Background())
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Live Data endpoint tests
// ---------------------------------------------------------------------------

func TestGetLiveDataBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/live_data/batch", r.URL.Path)
		fmt.Fprint(w, `{"live_datas":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetLiveDataBatch(context.Background(), GetLiveDataBatchParams{MilestoneIDs: "m1,m2"})
	require.NoError(t, err)
}

func TestGetLiveDataByMilestone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/live_data/milestone/m-1", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetLiveDataByMilestone(context.Background(), "m-1", GetLiveDataParams{})
	require.NoError(t, err)
}

func TestGetMilestoneGameStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/live_data/milestone/m-1/game_stats", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMilestoneGameStats(context.Background(), "m-1")
	require.NoError(t, err)
}

func TestGetLiveData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/live_data/scoreboard/milestone/m-1", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetLiveData(context.Background(), "scoreboard", "m-1", GetLiveDataParams{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Milestones endpoint tests
// ---------------------------------------------------------------------------

func TestGetMilestones(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/milestones", r.URL.Path)
		assert.Equal(t, "EVT", r.URL.Query().Get("event_ticker"))
		fmt.Fprint(w, `{"milestones":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMilestones(context.Background(), GetMilestonesParams{EventTicker: "EVT"})
	require.NoError(t, err)
}

func TestGetMilestone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/milestones/m-1", r.URL.Path)
		fmt.Fprint(w, `{"milestone":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMilestone(context.Background(), "m-1")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Multivariate Event Collections endpoint tests
// ---------------------------------------------------------------------------

func TestGetMultivariateEventCollections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/multivariate_event_collections", r.URL.Path)
		fmt.Fprint(w, `{"collections":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMultivariateEventCollections(context.Background(), GetMultivariateEventCollectionsParams{})
	require.NoError(t, err)
}

func TestGetMultivariateEventCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/multivariate_event_collections/COL-1", r.URL.Path)
		fmt.Fprint(w, `{"collection":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMultivariateEventCollection(context.Background(), "COL-1")
	require.NoError(t, err)
}

func TestGetMultivariateEventCollectionLookupHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/multivariate_event_collections/COL-1/lookup", r.URL.Path)
		fmt.Fprint(w, `{"lookup_history":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetMultivariateEventCollectionLookupHistory(context.Background(), "COL-1", GetMVECollectionLookupParams{})
	require.NoError(t, err)
}

func TestCreateMarketInMultivariateEventCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/multivariate_event_collections/COL-1", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.CreateMarketInMultivariateEventCollection(context.Background(), "COL-1", CreateMarketInMultivariateEventCollectionRequest{})
	require.NoError(t, err)
}

func TestLookupTickersForMarketInMultivariateEventCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/multivariate_event_collections/COL-1/lookup", r.URL.Path)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.LookupTickersForMarketInMultivariateEventCollection(context.Background(), "COL-1", LookupTickersForMarketInMultivariateEventCollectionRequest{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Structured Targets endpoint tests
// ---------------------------------------------------------------------------

func TestGetStructuredTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/structured_targets", r.URL.Path)
		fmt.Fprint(w, `{"structured_targets":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetStructuredTargets(context.Background(), GetStructuredTargetsParams{})
	require.NoError(t, err)
}

func TestGetStructuredTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/structured_targets/st-1", r.URL.Path)
		fmt.Fprint(w, `{"structured_target":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetStructuredTarget(context.Background(), "st-1")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// resolveCosts tests
// ---------------------------------------------------------------------------

func TestResolveCosts_ParameterizedPath(t *testing.T) {
	c := &Client{
		costPatterns: []endpointCostPattern{
			{
				method:  "DELETE",
				pattern: regexp.MustCompile(`^/trade-api/v2/portfolio/orders/[^/]+$`),
				cost:    20,
			},
			{
				method:  "GET",
				pattern: regexp.MustCompile(`^/trade-api/v2/markets/[^/]+$`),
				cost:    5,
			},
		},
		defaultCost: 10,
	}

	// Parameterized path should match the pattern
	read, write := c.resolveCosts("DELETE", "/trade-api/v2/portfolio/orders/abc-123-def", 1, 1)
	assert.Equal(t, 0.0, read, "DELETE should have zero read cost")
	assert.Equal(t, 20.0, write, "DELETE should use matched pattern cost")

	read, write = c.resolveCosts("GET", "/trade-api/v2/markets/KXBTC-100K", 1, 1)
	assert.Equal(t, 5.0, read, "GET should use matched pattern cost")
	assert.Equal(t, 0.0, write, "GET should have zero write cost")
}

func TestResolveCosts_DefaultCost(t *testing.T) {
	c := &Client{
		costPatterns: []endpointCostPattern{
			{
				method:  "GET",
				pattern: regexp.MustCompile(`^/trade-api/v2/markets$`),
				cost:    5,
			},
		},
		defaultCost: 10,
	}

	// Unmatched path falls back to defaultCost
	read, write := c.resolveCosts("GET", "/trade-api/v2/some/unknown/path", 1, 1)
	assert.Equal(t, 10.0, read)
	assert.Equal(t, 0.0, write)

	read, write = c.resolveCosts("POST", "/trade-api/v2/some/unknown/path", 1, 1)
	assert.Equal(t, 0.0, read)
	assert.Equal(t, 10.0, write)
}

func TestResolveCosts_NilPatterns(t *testing.T) {
	c := &Client{} // no costPatterns, no defaultCost

	// Should passthrough caller defaults
	read, write := c.resolveCosts("GET", "/trade-api/v2/exchange/status", 0.5, 0)
	assert.Equal(t, 0.5, read)
	assert.Equal(t, 0.0, write)
}

func TestNewClient_AutoConfigFailure(t *testing.T) {
	// Mock server that returns 500 for all requests
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"server error"}`)
	}))
	defer srv.Close()

	cfg := testClientConfig(t, srv.URL)
	c, err := NewClient(cfg)

	// Should succeed despite API failure — non-fatal fallback to defaults
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.NotNil(t, c.limiter, "should have default limiter")
}

func TestConfigureRateLimits_Concurrent(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/limits"):
			fmt.Fprint(w, `{"read":{"refill_rate":20,"bucket_capacity":20},"write":{"refill_rate":10,"bucket_capacity":10},"usage_tier":"standard"}`)
		case strings.Contains(r.URL.Path, "/endpoint_costs"):
			fmt.Fprint(w, `{"default_cost":10,"endpoint_costs":[{"method":"GET","path":"/markets","cost":5}]}`)
		default:
			fmt.Fprint(w, `{"status":"open"}`)
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)

	// Run ConfigureRateLimits and concurrent do() requests in parallel
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10; i++ {
			_ = c.ConfigureRateLimits(context.Background())
		}
	}()

	for i := 0; i < 10; i++ {
		_, _ = c.GetExchangeStatus(context.Background())
	}
	<-done
}
