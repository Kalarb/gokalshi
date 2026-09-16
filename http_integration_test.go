//go:build integration

package gokalshi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipOnAPIError skips the test if the error is an APIError with one of the given status codes.
// isAPIErrorCode reports whether err is an *APIError with the given status.
// Use it when a specific status is an expected outcome worth naming, rather
// than one more entry in a skipOnAPIError list.
func isAPIErrorCode(err error, code int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}

func skipOnAPIError(t *testing.T, err error, codes ...int) {
	t.Helper()
	if err == nil {
		return
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		for _, code := range codes {
			if apiErr.StatusCode == code {
				t.Skipf("endpoint returned %d — skipping", code)
			}
		}
	}
}

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

	t.Run("GetUserDataTimestamp", func(t *testing.T) {
		resp, err := c.GetUserDataTimestamp(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AsOfTime)
	})

	t.Run("GetSeriesFeeChanges", func(t *testing.T) {
		resp, err := c.GetSeriesFeeChanges(ctx, GetSeriesFeeChangesParams{})
		require.NoError(t, err)
		t.Logf("fee changes count=%d", len(resp.SeriesFeeChangeArr))
	})

	t.Run("GetIncentivePrograms", func(t *testing.T) {
		resp, err := c.GetIncentivePrograms(ctx)
		require.NoError(t, err)
		t.Logf("incentive programs count=%d", len(resp.IncentivePrograms))
	})
}

func TestHTTPIntegration_Markets(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

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

	t.Run("GetMarketOrderbooks", func(t *testing.T) {
		var tickers []string
		for _, m := range resp.Markets {
			tickers = append(tickers, m.Ticker)
			if len(tickers) >= 3 {
				break
			}
		}
		batchResp, err := c.GetMarketOrderbooks(ctx, GetMarketOrderbooksParams{
			Tickers: strings.Join(tickers, ","),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, batchResp.Orderbooks)
		t.Logf("fetched %d orderbooks", len(batchResp.Orderbooks))
	})

	t.Run("GetTrades", func(t *testing.T) {
		_, err := c.GetTrades(ctx, GetTradesParams{Ticker: ticker, Limit: 5})
		require.NoError(t, err)
	})

	t.Run("GetMarketCandlesticks", func(t *testing.T) {
		eventsResp, err := c.GetEvents(ctx, GetEventsParams{
			Status: "open", WithNestedMarkets: true, Limit: 1,
		})
		require.NoError(t, err)
		if len(eventsResp.Events) == 0 {
			t.Skip("no open events for candlestick test")
		}
		event := eventsResp.Events[0]
		if event.SeriesTicker == "" || len(event.Markets) == 0 {
			t.Skip("event missing series_ticker or nested markets")
		}
		marketTicker := event.Markets[0].Ticker

		now := time.Now().Unix()
		candleResp, err := c.GetMarketCandlesticks(ctx, event.SeriesTicker, marketTicker, GetMarketCandlesticksParams{
			StartTs: now - 3600, EndTs: now, PeriodInterval: 1,
		})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("market candlesticks: ticker=%s count=%d", candleResp.Ticker, len(candleResp.Candlesticks))
	})

	t.Run("GetBatchMarketCandlesticks", func(t *testing.T) {
		var tickers []string
		for _, m := range resp.Markets {
			tickers = append(tickers, m.Ticker)
			if len(tickers) >= 3 {
				break
			}
		}
		now := time.Now().Unix()
		batchResp, err := c.GetBatchMarketCandlesticks(ctx, GetBatchMarketCandlesticksParams{
			MarketTickers:  strings.Join(tickers, ","),
			StartTs:        now - 3600,
			EndTs:          now,
			PeriodInterval: 1,
		})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("batch candlesticks: %d markets returned", len(batchResp.Markets))
	})
}

func TestHTTPIntegration_Events(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	eventsResp, err := c.GetEvents(ctx, GetEventsParams{
		Status: "open", WithNestedMarkets: true, Limit: 5,
	})
	require.NoError(t, err)
	if len(eventsResp.Events) == 0 {
		t.Skip("no events available")
	}
	event := eventsResp.Events[0]
	eventTicker := event.EventTicker
	seriesTicker := event.SeriesTicker
	t.Logf("found %d events, using %s (series=%s)", len(eventsResp.Events), eventTicker, seriesTicker)

	t.Run("GetEvent", func(t *testing.T) {
		resp, err := c.GetEvent(ctx, eventTicker, GetEventParams{})
		require.NoError(t, err)
		assert.Equal(t, eventTicker, resp.Event.EventTicker)
	})

	t.Run("GetEventMetadata", func(t *testing.T) {
		_, err := c.GetEventMetadata(ctx, eventTicker)
		skipOnAPIError(t, err, 404)
		require.NoError(t, err)
	})

	t.Run("GetMultivariateEvents", func(t *testing.T) {
		resp, err := c.GetMultivariateEvents(ctx, GetMultivariateEventsParams{Limit: 5})
		require.NoError(t, err)
		assert.NotNil(t, resp.Events)
		t.Logf("multivariate events count=%d", len(resp.Events))
	})

	t.Run("GetEventCandlesticks", func(t *testing.T) {
		if seriesTicker == "" {
			t.Skip("event missing series_ticker")
		}
		now := time.Now().Unix()
		resp, err := c.GetEventCandlesticks(ctx, seriesTicker, eventTicker, GetEventCandlesticksParams{
			StartTs: now - 3600, EndTs: now, PeriodInterval: 1,
		})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("event candlesticks: %d market arrays", len(resp.MarketCandlesticks))
	})

	t.Run("GetEventForecastPercentileHistory", func(t *testing.T) {
		if seriesTicker == "" {
			t.Skip("event missing series_ticker")
		}
		now := time.Now().Unix()
		_, err := c.GetEventForecastPercentileHistory(ctx, seriesTicker, eventTicker, GetEventForecastPercentileHistoryParams{
			Percentiles:    "2500,5000,7500",
			StartTs:        now - 3600,
			EndTs:          now,
			PeriodInterval: 1,
		})
		// This endpoint returns 400 on DEMO for most events.
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetEventFeeChanges", func(t *testing.T) {
		resp, err := c.GetEventFeeChanges(ctx, GetEventFeeChangesParams{})
		require.NoError(t, err)
		t.Logf("event fee changes count=%d", len(resp.EventFeeChanges))
	})
}

func TestHTTPIntegration_Series(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	// Filter by category to avoid returning 10,000+ series (which times out).
	resp, err := c.GetSeriesList(ctx, GetSeriesListParams{Category: "Politics"})
	require.NoError(t, err)
	if len(resp.Series) == 0 {
		resp, err = c.GetSeriesList(ctx, GetSeriesListParams{})
		require.NoError(t, err)
		if len(resp.Series) == 0 {
			t.Skip("no series available")
		}
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

	t.Run("GetDeposits", func(t *testing.T) {
		_, err := c.GetDeposits(ctx, GetDepositsParams{})
		require.NoError(t, err)
	})

	t.Run("GetWithdrawals", func(t *testing.T) {
		_, err := c.GetWithdrawals(ctx, GetWithdrawalsParams{})
		require.NoError(t, err)
	})

	t.Run("GetPortfolioRestingOrderTotalValue", func(t *testing.T) {
		_, err := c.GetPortfolioRestingOrderTotalValue(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})
}

func TestHTTPIntegration_Orders(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	// Use a stable, far-future market to avoid flaky "market_closed" errors
	// from short-lived markets (e.g. MLB games) closing between discovery
	// and order placement.
	const ticker = "KXELONMARS-99"

	t.Run("GetOrders", func(t *testing.T) {
		resp, err := c.GetOrders(ctx, GetOrdersParams{Limit: 5})
		require.NoError(t, err)
		assert.NotNil(t, resp.Orders)
		t.Logf("orders count=%d", len(resp.Orders))
	})

	t.Run("CreateGetCancel", func(t *testing.T) {
		createResp, err := c.CreateOrderV2(ctx, newIntegrationOrder(ticker, "0.0100", "1.00"))
		require.NoError(t, err)
		orderID := createResp.OrderID
		assert.NotEmpty(t, orderID)
		t.Logf("created order %s on %s", orderID, ticker)
		t.Cleanup(func() { _, _ = c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		t.Run("GetOrder", func(t *testing.T) {
			// Retry — DEMO has up to 20s propagation delay on writes.
			var resp GetOrderResponse
			var getErr error
			for i := 0; i < 20; i++ {
				time.Sleep(1 * time.Second)
				resp, getErr = c.GetOrder(ctx, orderID)
				if getErr == nil {
					break
				}
				var apiErr *APIError
				if errors.As(getErr, &apiErr) && apiErr.StatusCode == 404 {
					continue
				}
				break
			}
			skipOnAPIError(t, getErr, 404)
			require.NoError(t, getErr)
			assert.Equal(t, orderID, resp.Order.OrderID)
		})

		t.Run("CancelOrder", func(t *testing.T) {
			resp, err := c.CancelOrderV2(ctx, orderID, CancelOrderV2Params{})
			require.NoError(t, err)
			assert.Equal(t, orderID, resp.OrderID)
			t.Logf("canceled order %s, reduced_by=%s", orderID, resp.ReducedBy)
		})
	})

	t.Run("AmendOrder", func(t *testing.T) {
		created, err := c.CreateOrderV2(ctx, newIntegrationOrder(ticker, "0.0100", "1.00"))
		require.NoError(t, err)
		orderID := created.OrderID
		t.Cleanup(func() { _, _ = c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		amended, err := c.AmendOrderV2(ctx, orderID, AmendOrderV2Request{
			Ticker: ticker,
			Side:   BookSideBid,
			Count:  "1.00",
			Price:  "0.0200",
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, amended.OrderID)
		t.Logf("amended order %s", amended.OrderID)
	})

	t.Run("DecreaseOrder", func(t *testing.T) {
		created, err := c.CreateOrderV2(ctx, newIntegrationOrder(ticker, "0.0100", "2.00"))
		require.NoError(t, err)
		orderID := created.OrderID
		t.Cleanup(func() { _, _ = c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		decreased, err := c.DecreaseOrderV2(ctx, orderID, DecreaseOrderV2Request{
			MarketTicker: ticker,
			ReduceTo:     ptr("1.00"),
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, decreased.OrderID)
		t.Logf("decreased order %s, remaining=%s", decreased.OrderID, decreased.RemainingCount)
	})

	t.Run("BatchCreateAndCancel", func(t *testing.T) {
		orders := []CreateOrderV2Request{
			newIntegrationOrder(ticker, "0.0100", "1.00"),
			newIntegrationOrder(ticker, "0.0100", "1.00"),
			newIntegrationOrder(ticker, "0.0100", "1.00"),
		}
		created, err := c.BatchCreateOrdersV2(ctx, BatchCreateOrdersV2Request{Orders: orders})
		require.NoError(t, err)

		// Batch V2 response entries are untyped in the spec, so the order id
		// has to be read out of the map.
		var orderIDs []string
		for _, entry := range created.Orders {
			if oid, ok := entry["order_id"].(string); ok && oid != "" {
				orderIDs = append(orderIDs, oid)
			}
		}
		assert.GreaterOrEqual(t, len(orderIDs), 1)
		t.Logf("batch created %d orders", len(orderIDs))

		t.Cleanup(func() {
			for _, oid := range orderIDs {
				_, _ = c.CancelOrderV2(context.Background(), oid, CancelOrderV2Params{})
			}
		})

		time.Sleep(2 * time.Second)

		cancels := make([]map[string]any, 0, len(orderIDs))
		for _, oid := range orderIDs {
			cancels = append(cancels, map[string]any{"order_id": oid})
		}
		_, err = c.BatchCancelOrdersV2(ctx, BatchCancelOrdersV2Request{Orders: cancels})
		require.NoError(t, err)
		t.Logf("batch canceled %d orders", len(orderIDs))
	})

	t.Run("GetQueuePositions", func(t *testing.T) {
		resp, err := c.GetQueuePositions(ctx, GetQueuePositionsParams{MarketTickers: ticker})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("queue positions count=%d", len(resp.QueuePositions))
	})

	t.Run("GetQueuePosition", func(t *testing.T) {
		created, err := c.CreateOrderV2(ctx, newIntegrationOrder(ticker, "0.0100", "1.00"))
		require.NoError(t, err)
		orderID := created.OrderID
		t.Cleanup(func() { _, _ = c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		qp, err := c.GetQueuePosition(ctx, orderID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		assert.NotEmpty(t, qp.QueuePositionFP)
		t.Logf("queue position for %s: %s", orderID, qp.QueuePositionFP)
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

	t.Run("GetAccountEndpointCosts", func(t *testing.T) {
		resp, err := c.GetAccountEndpointCosts(ctx)
		require.NoError(t, err)
		t.Logf("default_cost=%d endpoint_costs=%d", resp.DefaultCost, len(resp.EndpointCosts))
	})

	t.Run("GetAPIKeys", func(t *testing.T) {
		resp, err := c.GetAPIKeys(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("api keys count=%d", len(resp.APIKeys))
	})
}

func TestHTTPIntegration_Search(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetTagsByCategories", func(t *testing.T) {
		resp, err := c.GetTagsByCategories(ctx)
		require.NoError(t, err)
		assert.NotNil(t, resp.TagsByCategories)
		t.Logf("tags categories count=%d", len(resp.TagsByCategories))
	})

	t.Run("GetFiltersBySport", func(t *testing.T) {
		resp, err := c.GetFiltersBySport(ctx)
		require.NoError(t, err)
		assert.NotNil(t, resp.SportOrdering)
		t.Logf("sports count=%d", len(resp.SportOrdering))
	})
}
