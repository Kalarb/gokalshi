package gokalshi

// Query parameter structs for portfolio-related API endpoints.

// GetPositionsParams holds optional query parameters for GetPositions.
type GetPositionsParams struct {
	Ticker      string
	EventTicker string
	CountFilter string
	Limit       int
	Cursor      string
	Subaccount  int
}

func (p GetPositionsParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		String("event_ticker", p.EventTicker).
		String("count_filter", p.CountFilter).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int("subaccount", p.Subaccount).
		Build()
}

// GetFillsParams holds optional query parameters for GetFills.
type GetFillsParams struct {
	Ticker     string
	OrderID    string
	MinTs      int64
	MaxTs      int64
	Limit      int
	Cursor     string
	Subaccount int
}

func (p GetFillsParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		String("order_id", p.OrderID).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int("subaccount", p.Subaccount).
		Build()
}

// GetSettlementsParams holds optional query parameters for GetSettlements.
type GetSettlementsParams struct {
	Ticker      string
	EventTicker string
	MinTs       int64
	MaxTs       int64
	Limit       int
	Cursor      string
	Subaccount  int
}

func (p GetSettlementsParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		String("event_ticker", p.EventTicker).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int("subaccount", p.Subaccount).
		Build()
}
