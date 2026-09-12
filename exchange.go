package gokalshi

import (
	"context"
)

// GetExchangeStatus — Get Exchange Status
//
// GET /trade-api/v2/exchange/status
//
// Endpoint for getting the exchange status.
//
// See https://docs.kalshi.com/api-reference/exchange/get-exchange-status
func (c *Client) GetExchangeStatus(ctx context.Context) (ExchangeStatus, error) {
	return getJSON[ExchangeStatus](c, ctx, pathExchange+"/status", nil)
}

// GetExchangeSchedule — Get Exchange Schedule
//
// GET /trade-api/v2/exchange/schedule
//
// Endpoint for getting the exchange schedule.
//
// See https://docs.kalshi.com/api-reference/exchange/get-exchange-schedule
func (c *Client) GetExchangeSchedule(ctx context.Context) (GetExchangeScheduleResponse, error) {
	return getJSON[GetExchangeScheduleResponse](c, ctx, pathExchange+"/schedule", nil)
}

// GetUserDataTimestamp — Get User Data Timestamp
//
// GET /trade-api/v2/exchange/user_data_timestamp
//
// There is typically a short delay before exchange events are reflected in the
// API endpoints. Whenever possible, combine API responses to PUT/POST/DELETE
// requests with WebSocket data to obtain the most accurate view of the
// exchange state. This endpoint provides an approximate indication of when the
// data from the following endpoints was last validated: GetBalance,
// GetOrder(s), GetFills, GetPositions
//
// See https://docs.kalshi.com/api-reference/exchange/get-user-data-timestamp
func (c *Client) GetUserDataTimestamp(ctx context.Context) (GetUserDataTimestampResponse, error) {
	return getJSON[GetUserDataTimestampResponse](c, ctx, pathExchange+"/user_data_timestamp", nil)
}

// GetSeriesFeeChanges — Get Series Fee Changes
//
// GET /trade-api/v2/series/fee_changes
//
// See https://docs.kalshi.com/api-reference/exchange/get-series-fee-changes
func (c *Client) GetSeriesFeeChanges(ctx context.Context, params GetSeriesFeeChangesParams) (GetSeriesFeeChangesResponse, error) {
	return getJSON[GetSeriesFeeChangesResponse](c, ctx, pathSeries+"/fee_changes", params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetSeriesFeeChangesParams holds optional query parameters for GetSeriesFeeChanges.
type GetSeriesFeeChangesParams struct {
	SeriesTicker   string
	ShowHistorical bool
}

func (p GetSeriesFeeChangesParams) toMap() map[string]string {
	return NewQuery().
		String("series_ticker", p.SeriesTicker).
		Bool("show_historical", p.ShowHistorical).
		Build()
}
