package gokalshi

import (
	"context"
)

// GetBalance retrieves the account balance.
func (c *Client) GetBalance(ctx context.Context) (BalanceResponse, error) {
	return getJSON[BalanceResponse](c, ctx, pathPortfolio+"/balance", nil)
}

// GetPositions retrieves portfolio positions.
func (c *Client) GetPositions(ctx context.Context, params GetPositionsParams) (GetPositionsResponse, error) {
	return getJSON[GetPositionsResponse](c, ctx, pathPortfolio+"/positions", params.toMap())
}

// GetFills retrieves order fills with optional filtering and pagination.
func (c *Client) GetFills(ctx context.Context, params GetFillsParams) (GetFillsResponse, error) {
	return getJSON[GetFillsResponse](c, ctx, pathPortfolio+"/fills", params.toMap())
}

// GetSettlements retrieves settlement records.
func (c *Client) GetSettlements(ctx context.Context, params GetSettlementsParams) (GetSettlementsResponse, error) {
	return getJSON[GetSettlementsResponse](c, ctx, pathPortfolio+"/settlements", params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

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
