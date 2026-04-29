package gokalshi

// Query parameter structs for event-related API endpoints.

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
