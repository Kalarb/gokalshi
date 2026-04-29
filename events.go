package gokalshi

import (
	"context"
	"fmt"
)

const pathEvents = "/trade-api/v2/events"

// GetEvent retrieves details for a single event.
func (c *Client) GetEvent(ctx context.Context, eventTicker string, params GetEventParams) (GetEventResponse, error) {
	path := fmt.Sprintf("%s/%s", pathEvents, eventTicker)
	return getJSON[GetEventResponse](c, ctx, path, params.toMap())
}

// GetEvents retrieves events matching the given parameters.
func (c *Client) GetEvents(ctx context.Context, params GetEventsParams) (GetEventsResponse, error) {
	return getJSON[GetEventsResponse](c, ctx, pathEvents, params.toMap())
}

// GetEventMetadata retrieves metadata for a single event.
func (c *Client) GetEventMetadata(ctx context.Context, eventTicker string) (GetEventMetadataResponse, error) {
	path := fmt.Sprintf("%s/%s/metadata", pathEvents, eventTicker)
	return getJSON[GetEventMetadataResponse](c, ctx, path, nil)
}

// GetMultivariateEvents retrieves multivariate (combo) events.
func (c *Client) GetMultivariateEvents(ctx context.Context, params GetMultivariateEventsParams) (GetMultivariateEventsResponse, error) {
	return getJSON[GetMultivariateEventsResponse](c, ctx, pathEvents+"/multivariate", params.toMap())
}

// GetEventCandlesticks retrieves aggregated candlestick data for all markets in an event.
func (c *Client) GetEventCandlesticks(ctx context.Context, seriesTicker, eventTicker string, params GetEventCandlesticksParams) (GetEventCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/events/%s/candlesticks", seriesTicker, eventTicker)
	return getJSON[GetEventCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetEventForecastPercentileHistory retrieves historical forecast percentile data for an event.
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
// Response types
// ---------------------------------------------------------------------------

// EventDetail is the full event object returned by the Kalshi API.
type EventDetail struct {
	EventTicker          string               `json:"event_ticker"`
	SeriesTicker         string               `json:"series_ticker"`
	SubTitle             string               `json:"sub_title"`
	Title                string               `json:"title"`
	CollateralReturnType CollateralReturnType `json:"collateral_return_type"`
	MutuallyExclusive    bool                 `json:"mutually_exclusive"`
	AvailableOnBrokers   bool                 `json:"available_on_brokers"`
	ProductMetadata      any                  `json:"product_metadata"`
	Category             string               `json:"category"`
	StrikeDate           string               `json:"strike_date"`
	StrikePeriod         string               `json:"strike_period"`
	Markets              []MarketDetail       `json:"markets"`
	LastUpdatedTs        string               `json:"last_updated_ts"`
}

// Milestone is a milestone related to events.
type Milestone struct {
	ID                  string   `json:"id"`
	Category            string   `json:"category"`
	Type                string   `json:"type"`
	StartDate           string   `json:"start_date"`
	EndDate             string   `json:"end_date"`
	RelatedEventTickers []string `json:"related_event_tickers"`
	Title               string   `json:"title"`
	NotificationMessage string   `json:"notification_message"`
	Details             any      `json:"details"`
	PrimaryEventTickers []string `json:"primary_event_tickers"`
	LastUpdatedTs       string   `json:"last_updated_ts"`
	SourceID            string   `json:"source_id"`
	SourceIDs           any      `json:"source_ids"`
}

// GetEventResponse is the response from GET /events/{ticker}.
type GetEventResponse struct {
	Event   EventDetail    `json:"event"`
	Markets []MarketDetail `json:"markets"`
}

// GetEventsResponse is the paginated response from GET /events.
type GetEventsResponse struct {
	Events     []EventDetail `json:"events"`
	Cursor     string        `json:"cursor"`
	Milestones []Milestone   `json:"milestones"`
}

// MarketMetadata is metadata for a single market within an event.
type MarketMetadata struct {
	MarketTicker string `json:"market_ticker"`
	ImageURL     string `json:"image_url"`
	ColorCode    string `json:"color_code"`
}

// GetEventMetadataResponse is the response from GET /events/{ticker}/metadata.
type GetEventMetadataResponse struct {
	ImageURL          string             `json:"image_url"`
	MarketDetails     []MarketMetadata   `json:"market_details"`
	SettlementSources []SettlementSource `json:"settlement_sources"`
	FeaturedImageURL  string             `json:"featured_image_url"`
	Competition       string             `json:"competition"`
	CompetitionScope  string             `json:"competition_scope"`
}

// GetMultivariateEventsResponse is the response from GET /events/multivariate.
type GetMultivariateEventsResponse struct {
	Events []EventDetail `json:"events"`
	Cursor string        `json:"cursor"`
}

// GetEventCandlesticksResponse is the response from GET /series/{s}/events/{t}/candlesticks.
type GetEventCandlesticksResponse struct {
	MarketTickers      []string        `json:"market_tickers"`
	MarketCandlesticks [][]Candlestick `json:"market_candlesticks"`
	AdjustedEndTs      int64           `json:"adjusted_end_ts"`
}

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
