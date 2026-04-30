package gokalshi

import (
	"context"
	"fmt"
)

const pathEvents = "/trade-api/v2/events"

// GetEvent — Get Event
//
// GET /trade-api/v2/events/{event_ticker}
//
// Endpoint for getting data about an event by its ticker. An event represents
// a real-world occurrence that can be traded on, such as an election, sports
// game, or economic indicator release. Events contain one or more markets
// where users can place trades on different outcomes. All events are
// accessible through this endpoint, even if their associated markets are older
// than the historical cutoff.
//
// See https://trading-api.readme.io/reference/getevent
func (c *Client) GetEvent(ctx context.Context, eventTicker string, params GetEventParams) (GetEventResponse, error) {
	path := fmt.Sprintf("%s/%s", pathEvents, eventTicker)
	return getJSON[GetEventResponse](c, ctx, path, params.toMap())
}

// GetEvents — Get Events
//
// GET /trade-api/v2/events
//
// Get all events. This endpoint excludes multivariate events. To retrieve
// multivariate events, use the GET /events/multivariate endpoint. All events
// are accessible through this endpoint, even if their associated markets are
// older than the historical cutoff.
//
// See https://trading-api.readme.io/reference/getevents
func (c *Client) GetEvents(ctx context.Context, params GetEventsParams) (GetEventsResponse, error) {
	return getJSON[GetEventsResponse](c, ctx, pathEvents, params.toMap())
}

// GetEventMetadata — Get Event Metadata
//
// GET /trade-api/v2/events/{event_ticker}/metadata
//
// Endpoint for getting metadata about an event by its ticker. Returns only the
// metadata information for an event.
//
// See https://trading-api.readme.io/reference/geteventmetadata
func (c *Client) GetEventMetadata(ctx context.Context, eventTicker string) (GetEventMetadataResponse, error) {
	path := fmt.Sprintf("%s/%s/metadata", pathEvents, eventTicker)
	return getJSON[GetEventMetadataResponse](c, ctx, path, nil)
}

// GetMultivariateEvents — Get Multivariate Events
//
// GET /trade-api/v2/events/multivariate
//
// Retrieve multivariate (combo) events. These are dynamically created events
// from multivariate event collections. Supports filtering by series and
// collection ticker.
//
// See https://trading-api.readme.io/reference/getmultivariateevents
func (c *Client) GetMultivariateEvents(ctx context.Context, params GetMultivariateEventsParams) (GetMultivariateEventsResponse, error) {
	return getJSON[GetMultivariateEventsResponse](c, ctx, pathEvents+"/multivariate", params.toMap())
}

// GetEventCandlesticks — Get Event Candlesticks
//
// GET /trade-api/v2/series/{series_ticker}/events/{ticker}/candlesticks
//
// End-point for returning aggregated data across all markets corresponding to
// an event.
//
// See https://trading-api.readme.io/reference/getmarketcandlesticksbyevent
func (c *Client) GetEventCandlesticks(ctx context.Context, seriesTicker, eventTicker string, params GetEventCandlesticksParams) (GetEventCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/events/%s/candlesticks", seriesTicker, eventTicker)
	return getJSON[GetEventCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetEventForecastPercentileHistory — Get Event Forecast Percentile History
//
// GET /trade-api/v2/series/{series_ticker}/events/{ticker}/forecast_percentile_history
//
// Endpoint for getting the historical raw and formatted forecast numbers for
// an event at specific percentiles.
//
// See https://trading-api.readme.io/reference/geteventforecastpercentileshistory
func (c *Client) GetEventForecastPercentileHistory(ctx context.Context, seriesTicker, eventTicker string, params GetEventForecastPercentileHistoryParams) (GetEventForecastPercentileHistoryResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/events/%s/forecast_percentile_history", seriesTicker, eventTicker)
	return getJSON[GetEventForecastPercentileHistoryResponse](c, ctx, path, params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetEventParams holds optional query parameters for GetEvent.
type GetEventParams struct {
	WithNestedMarkets bool
}

func (p GetEventParams) toMap() map[string]string {
	return NewQuery().
		Bool("with_nested_markets", p.WithNestedMarkets).
		Build()
}

// GetEventsParams holds optional query parameters for GetEvents.
type GetEventsParams struct {
	Limit             int
	Cursor            string
	WithNestedMarkets bool
	WithMilestones    bool
	Status            MarketStatus
	SeriesTicker      string
	MinCloseTs        int64
	MinUpdatedTs      int64
}

func (p GetEventsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Bool("with_nested_markets", p.WithNestedMarkets).
		Bool("with_milestones", p.WithMilestones).
		String("status", string(p.Status)).
		String("series_ticker", p.SeriesTicker).
		Int64("min_close_ts", p.MinCloseTs).
		Int64("min_updated_ts", p.MinUpdatedTs).
		Build()
}

// GetMultivariateEventsParams holds optional query parameters for GetMultivariateEvents.
type GetMultivariateEventsParams struct {
	Limit             int
	Cursor            string
	SeriesTicker      string
	CollectionTicker  string
	WithNestedMarkets bool
}

func (p GetMultivariateEventsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		String("series_ticker", p.SeriesTicker).
		String("collection_ticker", p.CollectionTicker).
		Bool("with_nested_markets", p.WithNestedMarkets).
		Build()
}

// GetEventCandlesticksParams holds query parameters for GetEventCandlesticks.
type GetEventCandlesticksParams struct {
	StartTs        int64
	EndTs          int64
	PeriodInterval int // 1, 60, or 1440
}

func (p GetEventCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Build()
}

// GetEventForecastPercentileHistoryParams holds query parameters for forecast percentile history.
type GetEventForecastPercentileHistoryParams struct {
	Percentiles    string // comma-separated percentile values (0-10000)
	StartTs        int64
	EndTs          int64
	PeriodInterval int // 0, 1, 60, or 1440
}

func (p GetEventForecastPercentileHistoryParams) toMap() map[string]string {
	return NewQuery().
		String("percentiles", p.Percentiles).
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Build()
}

// ---------------------------------------------------------------------------
// Types not in types_generated.go
// ---------------------------------------------------------------------------

// ForecastPercentilePoint is a single percentile data point.
type ForecastPercentilePoint struct {
	Percentile           int     `json:"percentile"`
	RawNumericalForecast float64 `json:"raw_numerical_forecast"`
	NumericalForecast    float64 `json:"numerical_forecast"`
	FormattedForecast    string  `json:"formatted_forecast"`
}

// ForecastHistoryEntry is a single time-period forecast entry.
type ForecastHistoryEntry struct {
	EventTicker      string                    `json:"event_ticker"`
	EndPeriodTs      int64                     `json:"end_period_ts"`
	PeriodInterval   int                       `json:"period_interval"`
	PercentilePoints []ForecastPercentilePoint `json:"percentile_points"`
}

// GetEventForecastPercentileHistoryResponse is the response from
// GET /series/{s}/events/{t}/forecast_percentile_history.
type GetEventForecastPercentileHistoryResponse struct {
	ForecastHistory []ForecastHistoryEntry `json:"forecast_history"`
}
