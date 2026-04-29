package gokalshi

// Response types for event-related API endpoints.

// EventDetail is the full event object returned by the Kalshi API.
type EventDetail struct {
	EventTicker          string               `json:"event_ticker"`
	SeriesTicker         string               `json:"series_ticker"`
	SubTitle             string               `json:"sub_title"`
	Title                string               `json:"title"`
	CollateralReturnType CollateralReturnType  `json:"collateral_return_type"`
	MutuallyExclusive    bool                 `json:"mutually_exclusive"`
	AvailableOnBrokers   bool                 `json:"available_on_brokers"`
	ProductMetadata      any                  `json:"product_metadata"`
	Category             string               `json:"category"`
	StrikeDate           string               `json:"strike_date"`
	StrikePeriod         string               `json:"strike_period"`
	Markets              []MarketDetail        `json:"markets"`
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
