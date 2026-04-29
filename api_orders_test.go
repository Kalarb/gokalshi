package gokalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	_, err := c.BatchCancelOrders(context.Background(), []BatchCancelOrderEntry{
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
		Ticker: "TEST", Side: "yes", Action: "buy", CountFP: "5.00",
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
		ReduceByFP: "3.00",
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
