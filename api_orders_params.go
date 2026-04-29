package gokalshi

// Query parameter structs for order-related API endpoints.

// GetOrdersParams holds optional query parameters for GetOrders.
type GetOrdersParams struct {
	Ticker      string
	EventTicker string
	Status      OrderStatus
	Limit       int
	Cursor      string
	MinTs       int64
	MaxTs       int64
	Subaccount  int
}

func (p GetOrdersParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		String("event_ticker", p.EventTicker).
		String("status", string(p.Status)).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Int("subaccount", p.Subaccount).
		Build()
}

// GetQueuePositionsParams holds optional query parameters for GetQueuePositions.
type GetQueuePositionsParams struct {
	MarketTickers string
	EventTicker   string
	Subaccount    int
}

func (p GetQueuePositionsParams) toMap() map[string]string {
	return NewQuery().
		String("market_tickers", p.MarketTickers).
		String("event_ticker", p.EventTicker).
		Int("subaccount", p.Subaccount).
		Build()
}
