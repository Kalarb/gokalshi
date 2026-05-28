package gokalshi

import (
	"context"
)

// GetBalance — Get Balance
//
// GET /trade-api/v2/portfolio/balance
//
// Endpoint for getting the balance and portfolio value of a member. Both
// values are returned in cents.
//
// See https://trading-api.readme.io/reference/getbalance
func (c *Client) GetBalance(ctx context.Context) (GetBalanceResponse, error) {
	return getJSON[GetBalanceResponse](c, ctx, pathPortfolio+"/balance", nil)
}

// GetPositions — Get Positions
//
// GET /trade-api/v2/portfolio/positions
//
// Restricts the positions to those with any of following fields with non-zero
// values, as a comma separated list. The following values are accepted:
// position, total_traded
//
// See https://trading-api.readme.io/reference/getpositions
func (c *Client) GetPositions(ctx context.Context, params GetPositionsParams) (GetPositionsResponse, error) {
	return getJSON[GetPositionsResponse](c, ctx, pathPortfolio+"/positions", params.toMap())
}

// GetFills — Get Fills
//
// GET /trade-api/v2/portfolio/fills
//
// Endpoint for getting all fills for the member. A fill is when a trade you
// have is matched. Fills that occurred before the historical cutoff are only
// available via `GET /historical/fills`. See [Historical
// Data](https://docs.kalshi.com/getting_started/historical_data) for details.
//
// See https://trading-api.readme.io/reference/getfills
func (c *Client) GetFills(ctx context.Context, params GetFillsParams) (GetFillsResponse, error) {
	return getJSON[GetFillsResponse](c, ctx, pathPortfolio+"/fills", params.toMap())
}

// GetSettlements — Get Settlements
//
// GET /trade-api/v2/portfolio/settlements
//
// Endpoint for getting the member's settlements historical track.
//
// See https://trading-api.readme.io/reference/getsettlements
func (c *Client) GetSettlements(ctx context.Context, params GetSettlementsParams) (GetSettlementsResponse, error) {
	return getJSON[GetSettlementsResponse](c, ctx, pathPortfolio+"/settlements", params.toMap())
}

// GetDeposits — Get Deposits
//
// GET /trade-api/v2/portfolio/deposits
//
// Endpoint for getting the member's deposit history.
//
// See https://trading-api.readme.io/reference/getdeposits
func (c *Client) GetDeposits(ctx context.Context, params GetDepositsParams) (GetDepositsResponse, error) {
	return getJSON[GetDepositsResponse](c, ctx, pathPortfolio+"/deposits", params.toMap())
}

// GetWithdrawals — Get Withdrawals
//
// GET /trade-api/v2/portfolio/withdrawals
//
// Endpoint for getting the member's withdrawal history.
//
// See https://trading-api.readme.io/reference/getwithdrawals
func (c *Client) GetWithdrawals(ctx context.Context, params GetWithdrawalsParams) (GetWithdrawalsResponse, error) {
	return getJSON[GetWithdrawalsResponse](c, ctx, pathPortfolio+"/withdrawals", params.toMap())
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

// GetDepositsParams holds optional query parameters for GetDeposits.
type GetDepositsParams struct {
	Limit  int
	Cursor string
}

func (p GetDepositsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}

// GetWithdrawalsParams holds optional query parameters for GetWithdrawals.
type GetWithdrawalsParams struct {
	Limit  int
	Cursor string
}

func (p GetWithdrawalsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}
