package gokalshi

// Response types for portfolio-related API endpoints.

// BalanceResponse is the response from GET /portfolio/balance.
type BalanceResponse struct {
	Balance        int64 `json:"balance"`
	PortfolioValue int64 `json:"portfolio_value"`
	UpdatedTs      int64 `json:"updated_ts"`
}

// FillResponse is a single fill record from the Kalshi API.
type FillResponse struct {
	FillID           string `json:"fill_id"`
	TradeID          string `json:"trade_id"`
	OrderID          string `json:"order_id"`
	Ticker           string `json:"ticker"`
	MarketTicker     string `json:"market_ticker"`
	Side             Side   `json:"side"`
	Action           Action `json:"action"`
	CountFP          string `json:"count_fp"`
	YesPriceDollars  string `json:"yes_price_dollars"`
	NoPriceDollars   string `json:"no_price_dollars"`
	IsTaker          bool   `json:"is_taker"`
	FeeCost          string `json:"fee_cost"`
	CreatedTime      string `json:"created_time"`
	SubaccountNumber int    `json:"subaccount_number"`
	Ts               int64  `json:"ts"`
}

// GetFillsResponse is the paginated response from GET /portfolio/fills.
type GetFillsResponse struct {
	Fills  []FillResponse `json:"fills"`
	Cursor string         `json:"cursor"`
}

// MarketPositionResponse is a single market position from the Kalshi API.
type MarketPositionResponse struct {
	Ticker                string `json:"ticker"`
	TotalTradedDollars    string `json:"total_traded_dollars"`
	PositionFP            string `json:"position_fp"`
	MarketExposureDollars string `json:"market_exposure_dollars"`
	RealizedPnlDollars    string `json:"realized_pnl_dollars"`
	RestingOrdersCount    int    `json:"resting_orders_count"`
	FeesPaidDollars       string `json:"fees_paid_dollars"`
	LastUpdatedTs         string `json:"last_updated_ts"`
}

// EventPositionResponse is a single event position from the Kalshi API.
type EventPositionResponse struct {
	EventTicker          string `json:"event_ticker"`
	TotalCostDollars     string `json:"total_cost_dollars"`
	TotalCostSharesFP    string `json:"total_cost_shares_fp"`
	EventExposureDollars string `json:"event_exposure_dollars"`
	RealizedPnlDollars   string `json:"realized_pnl_dollars"`
	FeesPaidDollars      string `json:"fees_paid_dollars"`
}

// GetPositionsResponse is the paginated response from GET /portfolio/positions.
type GetPositionsResponse struct {
	MarketPositions []MarketPositionResponse `json:"market_positions"`
	EventPositions  []EventPositionResponse  `json:"event_positions"`
	Cursor          string                   `json:"cursor"`
}

// SettlementResponse is a single settlement record from the Kalshi API.
type SettlementResponse struct {
	Ticker              string       `json:"ticker"`
	EventTicker         string       `json:"event_ticker"`
	MarketResult        MarketResult `json:"market_result"`
	YesCountFP          string       `json:"yes_count_fp"`
	YesTotalCostDollars string       `json:"yes_total_cost_dollars"`
	NoCountFP           string       `json:"no_count_fp"`
	NoTotalCostDollars  string       `json:"no_total_cost_dollars"`
	Revenue             int          `json:"revenue"`
	SettledTime         string       `json:"settled_time"`
	FeeCost             string       `json:"fee_cost"`
	Value               *int         `json:"value"`
}

// GetSettlementsResponse is the paginated response from GET /portfolio/settlements.
type GetSettlementsResponse struct {
	Settlements []SettlementResponse `json:"settlements"`
	Cursor      string               `json:"cursor"`
}
