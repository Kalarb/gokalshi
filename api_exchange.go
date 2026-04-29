package gokalshi

import (
	"context"
)

const pathExchange = "/trade-api/v2/exchange"

// GetExchangeStatus returns the current exchange status.
func (c *Client) GetExchangeStatus(ctx context.Context) (GetExchangeStatusResponse, error) {
	return getJSON[GetExchangeStatusResponse](c, ctx, pathExchange+"/status", nil)
}

// GetExchangeAnnouncements returns all exchange-wide announcements.
func (c *Client) GetExchangeAnnouncements(ctx context.Context) (GetExchangeAnnouncementsResponse, error) {
	return getJSON[GetExchangeAnnouncementsResponse](c, ctx, pathExchange+"/announcements", nil)
}

// GetExchangeSchedule returns the exchange trading schedule.
func (c *Client) GetExchangeSchedule(ctx context.Context) (GetExchangeScheduleResponse, error) {
	return getJSON[GetExchangeScheduleResponse](c, ctx, pathExchange+"/schedule", nil)
}

// GetUserDataTimestamp returns when user data was last updated.
func (c *Client) GetUserDataTimestamp(ctx context.Context) (GetUserDataTimestampResponse, error) {
	return getJSON[GetUserDataTimestampResponse](c, ctx, pathExchange+"/user_data_timestamp", nil)
}

// GetSeriesFeeChanges returns fee change records for series.
func (c *Client) GetSeriesFeeChanges(ctx context.Context, params GetSeriesFeeChangesParams) (GetSeriesFeeChangesResponse, error) {
	return getJSON[GetSeriesFeeChangesResponse](c, ctx, pathSeries+"/fee_changes", params.toMap())
}
