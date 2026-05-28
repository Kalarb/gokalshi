package gokalshi

import (
	"context"
	"fmt"
)

// GetHistoricalCutoff — Get Historical Cutoff Timestamps
//
// GET /trade-api/v2/historical/cutoff
//
// Returns the cutoff timestamps that define the boundary between **live** and
// **historical** data.
//
// See https://trading-api.readme.io/reference/gethistoricalcutoff
func (c *Client) GetHistoricalCutoff(ctx context.Context) (GetHistoricalCutoffResponse, error) {
	return getJSON[GetHistoricalCutoffResponse](c, ctx, pathHistorical+"/cutoff", nil)
}

// GetHistoricalFills — Get Historical Fills
//
// GET /trade-api/v2/historical/fills
func (c *Client) GetHistoricalFills(ctx context.Context, params GetHistoricalFillsParams) (GetFillsResponse, error) {
	return getJSON[GetFillsResponse](c, ctx, pathHistorical+"/fills", params.toMap())
}

// GetHistoricalOrders — Get Historical Orders
//
// GET /trade-api/v2/historical/orders
//
// Endpoint for getting orders that have been archived to the historical
// database.
//
// See https://trading-api.readme.io/reference/gethistoricalorders
func (c *Client) GetHistoricalOrders(ctx context.Context, params GetHistoricalOrdersParams) (GetOrdersResponse, error) {
	return getJSON[GetOrdersResponse](c, ctx, pathHistorical+"/orders", params.toMap())
}

// GetHistoricalTrades — Get Historical Trades
//
// GET /trade-api/v2/historical/trades
func (c *Client) GetHistoricalTrades(ctx context.Context, params GetHistoricalTradesParams) (GetTradesResponse, error) {
	return getJSON[GetTradesResponse](c, ctx, pathHistorical+"/trades", params.toMap())
}

// GetHistoricalMarkets — Get Historical Markets
//
// GET /trade-api/v2/historical/markets
//
// Endpoint for getting markets that have been archived to the historical
// database. Filters are mutually exclusive.
//
// See https://trading-api.readme.io/reference/gethistoricalmarkets
func (c *Client) GetHistoricalMarkets(ctx context.Context, params GetHistoricalMarketsParams) (GetMarketsResponse, error) {
	return getJSON[GetMarketsResponse](c, ctx, pathHistorical+"/markets", params.toMap())
}

// GetHistoricalMarket — Get Historical Market
//
// GET /trade-api/v2/historical/markets/{ticker}
//
// Endpoint for getting data about a specific market by its ticker from the
// historical database.
//
// See https://trading-api.readme.io/reference/gethistoricalmarket
func (c *Client) GetHistoricalMarket(ctx context.Context, ticker string) (GetMarketResponse, error) {
	path := fmt.Sprintf("%s/markets/%s", pathHistorical, ticker)
	return getJSON[GetMarketResponse](c, ctx, path, nil)
}

// GetHistoricalMarketCandlesticks — Get Historical Market Candlesticks
//
// GET /trade-api/v2/historical/markets/{ticker}/candlesticks
func (c *Client) GetHistoricalMarketCandlesticks(ctx context.Context, ticker string, params GetHistoricalMarketCandlesticksParams) (GetMarketCandlesticksHistoricalResponse, error) {
	path := fmt.Sprintf("%s/markets/%s/candlesticks", pathHistorical, ticker)
	return getJSON[GetMarketCandlesticksHistoricalResponse](c, ctx, path, params.toMap())
}

// GetHistoricalFillsParams are query parameters for GetHistoricalFills.
type GetHistoricalFillsParams struct {
	Cursor string
	Limit  int
	Ticker string
}

func (p GetHistoricalFillsParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		String("ticker", p.Ticker).
		Build()
}

// GetHistoricalOrdersParams are query parameters for GetHistoricalOrders.
type GetHistoricalOrdersParams struct {
	Cursor string
	Limit  int
	Ticker string
}

func (p GetHistoricalOrdersParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		String("ticker", p.Ticker).
		Build()
}

// GetHistoricalTradesParams are query parameters for GetHistoricalTrades.
type GetHistoricalTradesParams struct {
	Cursor string
	Limit  int
	Ticker string
}

func (p GetHistoricalTradesParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		String("ticker", p.Ticker).
		Build()
}

// GetHistoricalMarketsParams are query parameters for GetHistoricalMarkets.
type GetHistoricalMarketsParams struct {
	Cursor string
	Limit  int
}

func (p GetHistoricalMarketsParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		Build()
}

// GetHistoricalMarketCandlesticksParams are query parameters for GetHistoricalMarketCandlesticks.
type GetHistoricalMarketCandlesticksParams struct {
	StartTS        int64
	EndTS          int64
	PeriodInterval int
}

func (p GetHistoricalMarketCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		Int64("start_ts", p.StartTS).
		Int64("end_ts", p.EndTS).
		Int("period_interval", p.PeriodInterval).
		Build()
}
