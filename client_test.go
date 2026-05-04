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

// newTestClient creates a Client with a fast rate limiter and no retry delay.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	cfg := testClientConfig(t, serverURL)
	limiter := NewReadWriteTokenBucket(TokenBucketConfig{
		ReadRate: 100, WriteRate: 100,
		WindowSize: 1.0, SafetyPadding: 0,
	})
	return NewClient(cfg, WithRateLimiter(limiter), WithBaseDelay(1*time.Millisecond))
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
