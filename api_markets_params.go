package gokalshi

// Query parameter structs for market-related API endpoints.

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
