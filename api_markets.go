package gokalshi

import (
	"context"
	"fmt"
)

const pathMarkets = "/trade-api/v2/markets"

// GetMarketOrderbook retrieves the orderbook for a market.
func (c *Client) GetMarketOrderbook(ctx context.Context, ticker string, params GetOrderbookParams) (GetMarketOrderbookResponse, error) {
	path := fmt.Sprintf("%s/%s/orderbook", pathMarkets, ticker)
	return getJSON[GetMarketOrderbookResponse](c, ctx, path, params.toMap())
}

// GetMarketOrderbooks retrieves orderbooks for multiple markets in a single request.
func (c *Client) GetMarketOrderbooks(ctx context.Context, params GetMarketOrderbooksParams) (GetMarketOrderbooksResponse, error) {
	return getJSON[GetMarketOrderbooksResponse](c, ctx, pathMarkets+"/orderbooks", params.toMap())
}

// GetTrades retrieves recent trades.
func (c *Client) GetTrades(ctx context.Context, params GetTradesParams) (GetTradesResponse, error) {
	return getJSON[GetTradesResponse](c, ctx, pathMarkets+"/trades", params.toMap())
}

// GetMarket retrieves details for a single market.
func (c *Client) GetMarket(ctx context.Context, ticker string) (MarketResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMarkets, ticker)
	return getJSON[MarketResponse](c, ctx, path, nil)
}

// GetMarkets retrieves markets matching the given parameters.
func (c *Client) GetMarkets(ctx context.Context, params GetMarketsParams) (GetMarketsResponse, error) {
	return getJSON[GetMarketsResponse](c, ctx, pathMarkets, params.toMap())
}

// GetMarketCandlesticks retrieves candlestick data for a single market.
func (c *Client) GetMarketCandlesticks(ctx context.Context, seriesTicker, ticker string, params GetMarketCandlesticksParams) (GetMarketCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/markets/%s/candlesticks", seriesTicker, ticker)
	return getJSON[GetMarketCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetBatchMarketCandlesticks retrieves candlestick data for multiple markets.
func (c *Client) GetBatchMarketCandlesticks(ctx context.Context, params GetBatchMarketCandlesticksParams) (GetBatchMarketCandlesticksResponse, error) {
	return getJSON[GetBatchMarketCandlesticksResponse](c, ctx, pathMarkets+"/candlesticks", params.toMap())
}
