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

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// GetExchangeStatusResponse is the response from GET /exchange/status.
type GetExchangeStatusResponse struct {
	ExchangeActive              bool   `json:"exchange_active"`
	TradingActive               bool   `json:"trading_active"`
	ExchangeEstimatedResumeTime string `json:"exchange_estimated_resume_time"`
}

// AnnouncementResponse is a single exchange announcement.
type AnnouncementResponse struct {
	Type         AnnouncementType   `json:"type"`
	Message      string             `json:"message"`
	DeliveryTime string             `json:"delivery_time"`
	Status       AnnouncementStatus `json:"status"`
}

// GetExchangeAnnouncementsResponse is the response from GET /exchange/announcements.
type GetExchangeAnnouncementsResponse struct {
	Announcements []AnnouncementResponse `json:"announcements"`
}

// SeriesFeeChange is a single fee change record.
type SeriesFeeChange struct {
	ID            string  `json:"id"`
	SeriesTicker  string  `json:"series_ticker"`
	FeeType       FeeType `json:"fee_type"`
	FeeMultiplier float64 `json:"fee_multiplier"`
	ScheduledTs   string  `json:"scheduled_ts"`
}

// GetSeriesFeeChangesResponse is the response from GET /series/fee_changes.
type GetSeriesFeeChangesResponse struct {
	SeriesFeeChangeArr []SeriesFeeChange `json:"series_fee_change_arr"`
}

// TradingSession is a single open/close time window.
type TradingSession struct {
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// StandardHoursWeek is the weekly trading schedule.
type StandardHoursWeek struct {
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	Monday    []TradingSession `json:"monday"`
	Tuesday   []TradingSession `json:"tuesday"`
	Wednesday []TradingSession `json:"wednesday"`
	Thursday  []TradingSession `json:"thursday"`
	Friday    []TradingSession `json:"friday"`
	Saturday  []TradingSession `json:"saturday"`
	Sunday    []TradingSession `json:"sunday"`
}

// MaintenanceWindow is a scheduled maintenance period.
type MaintenanceWindow struct {
	StartDatetime string `json:"start_datetime"`
	EndDatetime   string `json:"end_datetime"`
}

// ExchangeSchedule is the full exchange schedule.
type ExchangeSchedule struct {
	StandardHours      []StandardHoursWeek `json:"standard_hours"`
	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows"`
}

// GetExchangeScheduleResponse is the response from GET /exchange/schedule.
type GetExchangeScheduleResponse struct {
	Schedule ExchangeSchedule `json:"schedule"`
}

// GetUserDataTimestampResponse is the response from GET /exchange/user_data_timestamp.
type GetUserDataTimestampResponse struct {
	AsOfTime string `json:"as_of_time"`
}
