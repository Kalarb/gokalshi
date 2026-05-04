//go:build integration

package gokalshi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPIntegration_Exchange(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetExchangeStatus", func(t *testing.T) {
		resp, err := c.GetExchangeStatus(ctx)
		require.NoError(t, err)
		t.Logf("exchange_active=%v trading_active=%v", resp.ExchangeActive, resp.TradingActive)
	})

	t.Run("GetExchangeSchedule", func(t *testing.T) {
		resp, err := c.GetExchangeSchedule(ctx)
		require.NoError(t, err)
		assert.NotNil(t, resp.Schedule.StandardHours)
	})

	t.Run("GetExchangeAnnouncements", func(t *testing.T) {
		_, err := c.GetExchangeAnnouncements(ctx)
		require.NoError(t, err)
	})

	t.Run("GetUserDataTimestamp", func(t *testing.T) {
		resp, err := c.GetUserDataTimestamp(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AsOfTime)
	})
}

func TestHTTPIntegration_Markets(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetMarkets", func(t *testing.T) {
		resp, err := c.GetMarkets(ctx, GetMarketsParams{Status: "open", Limit: 5})
		require.NoError(t, err)
		if len(resp.Markets) == 0 {
			t.Skip("no active markets available")
		}
		ticker := resp.Markets[0].Ticker
		t.Logf("found %d markets, using %s", len(resp.Markets), ticker)

		t.Run("GetMarket", func(t *testing.T) {
			resp, err := c.GetMarket(ctx, ticker)
			require.NoError(t, err)
			assert.Equal(t, ticker, resp.Market.Ticker)
		})

		t.Run("GetMarketOrderbook", func(t *testing.T) {
			resp, err := c.GetMarketOrderbook(ctx, ticker, GetOrderbookParams{})
			require.NoError(t, err)
			assert.NotNil(t, resp.OrderbookFP)
		})

		t.Run("GetTrades", func(t *testing.T) {
			_, err := c.GetTrades(ctx, GetTradesParams{Ticker: ticker, Limit: 5})
			require.NoError(t, err)
		})
	})
}

func TestHTTPIntegration_Events(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	resp, err := c.GetEvents(ctx, GetEventsParams{Limit: 5})
	require.NoError(t, err)
	if len(resp.Events) == 0 {
		t.Skip("no events available")
	}
	ticker := resp.Events[0].EventTicker
	t.Logf("found %d events, using %s", len(resp.Events), ticker)

	t.Run("GetEvent", func(t *testing.T) {
		resp, err := c.GetEvent(ctx, ticker, GetEventParams{})
		require.NoError(t, err)
		assert.Equal(t, ticker, resp.Event.EventTicker)
	})

	t.Run("GetEventMetadata", func(t *testing.T) {
		_, err := c.GetEventMetadata(ctx, ticker)
		// May return 404 for some events — that's ok.
		if err != nil {
			var apiErr *APIError
			if assert.ErrorAs(t, err, &apiErr) && apiErr.StatusCode == 404 {
				t.Skip("event metadata not available for this event")
			}
		}
	})
}

func TestHTTPIntegration_Series(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	resp, err := c.GetSeriesList(ctx, GetSeriesListParams{})
	require.NoError(t, err)
	if len(resp.Series) == 0 {
		t.Skip("no series available")
	}
	ticker := resp.Series[0].Ticker
	t.Logf("found %d series, using %s", len(resp.Series), ticker)

	t.Run("GetSeries", func(t *testing.T) {
		resp, err := c.GetSeries(ctx, ticker, GetSeriesParams{})
		require.NoError(t, err)
		assert.Equal(t, ticker, resp.Series.Ticker)
	})
}

func TestHTTPIntegration_Portfolio(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetBalance", func(t *testing.T) {
		resp, err := c.GetBalance(ctx)
		require.NoError(t, err)
		t.Logf("balance=%d", resp.Balance)
	})

	t.Run("GetPositions", func(t *testing.T) {
		_, err := c.GetPositions(ctx, GetPositionsParams{})
		require.NoError(t, err)
	})

	t.Run("GetFills", func(t *testing.T) {
		_, err := c.GetFills(ctx, GetFillsParams{})
		require.NoError(t, err)
	})

	t.Run("GetSettlements", func(t *testing.T) {
		_, err := c.GetSettlements(ctx, GetSettlementsParams{})
		require.NoError(t, err)
	})
}

func TestHTTPIntegration_Orders(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	// Find an active market to place a test order on.
	markets, err := c.GetMarkets(ctx, GetMarketsParams{Status: "open", Limit: 5})
	require.NoError(t, err)
	if len(markets.Markets) == 0 {
		t.Skip("no active markets for order test")
	}
	ticker := markets.Markets[0].Ticker

	// Place at 1c — won't fill.
	createResp, err := c.CreateOrder(ctx, CreateOrderRequest{
		Ticker:          ticker,
		Side:            SideYes,
		Action:          ActionBuy,
		CountFP:         "1.00",
		YesPriceDollars: "0.0100",
		TimeInForce:     TimeInForceGTC,
	})
	require.NoError(t, err)
	orderID := createResp.Order.OrderID
	assert.NotEmpty(t, orderID)
	t.Logf("created order %s on %s", orderID, ticker)

	// Best-effort cleanup if test fails before CancelOrder.
	t.Cleanup(func() { c.CancelOrder(context.Background(), orderID) })

	// Small delay for propagation.
	time.Sleep(1 * time.Second)

	t.Run("GetOrder", func(t *testing.T) {
		resp, err := c.GetOrder(ctx, orderID)
		require.NoError(t, err)
		assert.Equal(t, orderID, resp.Order.OrderID)
	})

	t.Run("CancelOrder", func(t *testing.T) {
		resp, err := c.CancelOrder(ctx, orderID)
		require.NoError(t, err)
		assert.Equal(t, orderID, resp.Order.OrderID)
		t.Logf("canceled order %s, status=%s", orderID, resp.Order.Status)
	})
}

func TestHTTPIntegration_Account(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetAccountAPILimits", func(t *testing.T) {
		resp, err := c.GetAccountAPILimits(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.UsageTier)
		t.Logf("tier=%s read=%d write=%d", resp.UsageTier, resp.Read.RefillRate, resp.Write.RefillRate)
	})
}
