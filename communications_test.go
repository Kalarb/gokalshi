package gokalshi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCommunicationsID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/communications/id", r.URL.Path)
		fmt.Fprint(w, `{"communications_id":"comm-123"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.GetCommunicationsID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "comm-123", resp.CommunicationsID)
}

func TestCreateRFQ(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/rfqs", r.URL.Path)
		fmt.Fprint(w, `{"id":"rfq-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.CreateRFQ(context.Background(), CreateRFQRequest{MarketTicker: "TICK"})
	require.NoError(t, err)
	assert.Equal(t, "rfq-1", resp.ID)
}

func TestGetRFQs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/communications/rfqs", r.URL.Path)
		assert.Equal(t, "TICK", r.URL.Query().Get("market_ticker"))
		fmt.Fprint(w, `{"rfqs":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetRFQs(context.Background(), GetRFQsParams{Ticker: "TICK"})
	require.NoError(t, err)
}

func TestGetRFQ(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/communications/rfqs/rfq-1", r.URL.Path)
		fmt.Fprint(w, `{"rfq":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetRFQ(context.Background(), "rfq-1")
	require.NoError(t, err)
}

func TestDeleteRFQ(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/rfqs/rfq-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteRFQ(context.Background(), "rfq-1")
	require.NoError(t, err)
}

func TestCreateQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/quotes", r.URL.Path)
		fmt.Fprint(w, `{"id":"q-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.CreateQuote(context.Background(), CreateQuoteRequest{RFQID: "rfq-1"})
	require.NoError(t, err)
	assert.Equal(t, "q-1", resp.ID)
}

func TestGetQuotes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/communications/quotes", r.URL.Path)
		fmt.Fprint(w, `{"quotes":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetQuotes(context.Background(), GetQuotesParams{})
	require.NoError(t, err)
}

func TestGetQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/communications/quotes/q-1", r.URL.Path)
		fmt.Fprint(w, `{"quote":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetQuote(context.Background(), "q-1")
	require.NoError(t, err)
}

func TestDeleteQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/quotes/q-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteQuote(context.Background(), "q-1")
	require.NoError(t, err)
}

func TestAcceptQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/quotes/q-1/accept", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.AcceptQuote(context.Background(), "q-1", AcceptQuoteRequest{AcceptedSide: SideYes})
	require.NoError(t, err)
}

func TestConfirmQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/communications/quotes/q-1/confirm", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.ConfirmQuote(context.Background(), "q-1")
	require.NoError(t, err)
}
