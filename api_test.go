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
