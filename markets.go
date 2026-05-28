package gokalshi

import (
	"context"
	"fmt"
)

const pathMarkets = "/trade-api/v2/markets"

// GetMarketOrderbook — Get Market Orderbook
//
// GET /trade-api/v2/markets/{ticker}/orderbook
//
// Endpoint for getting the current order book for a specific market. The order
// book shows all active bid orders for both yes and no sides of a binary
// market. It returns yes bids and no bids only (no asks are returned). This is
// because in binary markets, a bid for yes at price X is equivalent to an ask
// for no at price (100-X). For example, a yes bid at 7¢ is the same as a no
// ask at 93¢, with identical contract sizes. Each side shows price levels
// with their corresponding quantities and order counts, organized from best to
// worst prices.
//
// See https://trading-api.readme.io/reference/getmarketorderbook
func (c *Client) GetMarketOrderbook(ctx context.Context, ticker string, params GetOrderbookParams) (GetMarketOrderbookResponse, error) {
	path := fmt.Sprintf("%s/%s/orderbook", pathMarkets, ticker)
	return getJSON[GetMarketOrderbookResponse](c, ctx, path, params.toMap())
}

// GetMarketOrderbooks — Get Multiple Market Orderbooks
//
// GET /trade-api/v2/markets/orderbooks
//
// Endpoint for getting the current order books for multiple markets in a
// single request. The order book shows all active bid orders for both yes and
// no sides of a binary market. It returns yes bids and no bids only (no asks
// are returned). This is because in binary markets, a bid for yes at price X
// is equivalent to an ask for no at price (100-X). For example, a yes bid at
// 7¢ is the same as a no ask at 93¢, with identical contract sizes. Each
// side shows price levels with their corresponding quantities and order
// counts, organized from best to worst prices. Returns one orderbook per
// requested market ticker.
//
// See https://trading-api.readme.io/reference/getmarketorderbooks
func (c *Client) GetMarketOrderbooks(ctx context.Context, params GetMarketOrderbooksParams) (GetMarketOrderbooksResponse, error) {
	return getJSON[GetMarketOrderbooksResponse](c, ctx, pathMarkets+"/orderbooks", params.toMap())
}

// GetTrades — Get Trades
//
// GET /trade-api/v2/markets/trades
//
// Endpoint for getting all trades for all markets. A trade represents a
// completed transaction between two users on a specific market. Each trade
// includes the market ticker, price, quantity, and timestamp information. This
// endpoint returns a paginated response. Use the 'limit' parameter to control
// page size (1-1000, defaults to 100). The response includes a 'cursor' field
// - pass this value in the 'cursor' parameter of your next request to get the
// next page. An empty cursor indicates no more pages are available.
//
// See https://trading-api.readme.io/reference/gettrades
func (c *Client) GetTrades(ctx context.Context, params GetTradesParams) (GetTradesResponse, error) {
	return getJSON[GetTradesResponse](c, ctx, pathMarkets+"/trades", params.toMap())
}

// GetMarket — Get Market
//
// GET /trade-api/v2/markets/{ticker}
//
// Endpoint for getting data about a specific market by its ticker. A market
// represents a specific binary outcome within an event that users can trade on
// (e.g., "Will candidate X win?"). Markets have yes/no positions, current
// prices, volume, and settlement rules.
//
// See https://trading-api.readme.io/reference/getmarket
func (c *Client) GetMarket(ctx context.Context, ticker string) (GetMarketResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMarkets, ticker)
	return getJSON[GetMarketResponse](c, ctx, path, nil)
}

// GetMarkets — Get Markets
//
// GET /trade-api/v2/markets
//
// Filter by market status. Possible values: `unopened`, `open`, `closed`,
// `settled`. Leave empty to return markets with any status. - Only one
// `status` filter may be supplied at a time. - Timestamp filters will be
// mutually exclusive from other timestamp filters and certain status filters.
//
// See https://trading-api.readme.io/reference/getmarkets
func (c *Client) GetMarkets(ctx context.Context, params GetMarketsParams) (GetMarketsResponse, error) {
	return getJSON[GetMarketsResponse](c, ctx, pathMarkets, params.toMap())
}

// GetMarketCandlesticks — Get Market Candlesticks
//
// GET /trade-api/v2/series/{series_ticker}/markets/{ticker}/candlesticks
//
// Time period length of each candlestick in minutes. Valid values: 1 (1
// minute), 60 (1 hour), 1440 (1 day). Candlesticks for markets that settled
// before the historical cutoff are only available via `GET
// /historical/markets/{ticker}/candlesticks`. See [Historical
// Data](https://docs.kalshi.com/getting_started/historical_data) for details.
//
// See https://trading-api.readme.io/reference/getmarketcandlesticks
func (c *Client) GetMarketCandlesticks(ctx context.Context, seriesTicker, ticker string, params GetMarketCandlesticksParams) (GetMarketCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/markets/%s/candlesticks", seriesTicker, ticker)
	return getJSON[GetMarketCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetBatchMarketCandlesticks — Batch Get Market Candlesticks
//
// GET /trade-api/v2/markets/candlesticks
//
// Endpoint for retrieving candlestick data for multiple markets.
//
// See https://trading-api.readme.io/reference/batchgetmarketcandlesticks
func (c *Client) GetBatchMarketCandlesticks(ctx context.Context, params GetBatchMarketCandlesticksParams) (BatchGetMarketCandlesticksResponse, error) {
	return getJSON[BatchGetMarketCandlesticksResponse](c, ctx, pathMarkets+"/candlesticks", params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetOrderbookParams holds optional query parameters for GetMarketOrderbook.
type GetOrderbookParams struct {
	Depth int
}

func (p GetOrderbookParams) toMap() map[string]string {
	return NewQuery().
		Int("depth", p.Depth).
		Build()
}

// GetMarketOrderbooksParams holds query parameters for GetMarketOrderbooks.
type GetMarketOrderbooksParams struct {
	Tickers string // comma-separated, 1-100 tickers
}

func (p GetMarketOrderbooksParams) toMap() map[string]string {
	return NewQuery().
		String("tickers", p.Tickers).
		Build()
}

// GetTradesParams holds optional query parameters for GetTrades.
type GetTradesParams struct {
	Ticker string
	Limit  int
	Cursor string
	MinTs  int64
	MaxTs  int64
}

func (p GetTradesParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Build()
}

// GetMarketsParams holds optional query parameters for GetMarkets.
type GetMarketsParams struct {
	Limit        int
	Cursor       string
	EventTicker  string
	SeriesTicker string
	Status       MarketStatus
	Tickers      string
	MVEFilter    string
	MinCreatedTs int64
	MaxCreatedTs int64
	MinUpdatedTs int64
	MinCloseTs   int64
	MaxCloseTs   int64
	MinSettledTs int64
	MaxSettledTs int64
}

func (p GetMarketsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		String("event_ticker", p.EventTicker).
		String("series_ticker", p.SeriesTicker).
		String("status", string(p.Status)).
		String("tickers", p.Tickers).
		String("mve_filter", p.MVEFilter).
		Int64("min_created_ts", p.MinCreatedTs).
		Int64("max_created_ts", p.MaxCreatedTs).
		Int64("min_updated_ts", p.MinUpdatedTs).
		Int64("min_close_ts", p.MinCloseTs).
		Int64("max_close_ts", p.MaxCloseTs).
		Int64("min_settled_ts", p.MinSettledTs).
		Int64("max_settled_ts", p.MaxSettledTs).
		Build()
}

// GetMarketCandlesticksParams holds query parameters for GetCandlesticks.
type GetMarketCandlesticksParams struct {
	StartTs                  int64
	EndTs                    int64
	PeriodInterval           int // 1, 60, or 1440
	IncludeLatestBeforeStart bool
}

func (p GetMarketCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Bool("include_latest_before_start", p.IncludeLatestBeforeStart).
		Build()
}

// GetBatchMarketCandlesticksParams holds query parameters for GetBatchCandlesticks.
type GetBatchMarketCandlesticksParams struct {
	MarketTickers            string // comma-separated, max 100
	StartTs                  int64
	EndTs                    int64
	PeriodInterval           int
	IncludeLatestBeforeStart bool
}

func (p GetBatchMarketCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		String("market_tickers", p.MarketTickers).
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Bool("include_latest_before_start", p.IncludeLatestBeforeStart).
		Build()
}
