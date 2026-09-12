package gokalshi

import "context"

// FCM endpoints are available only to Futures Commission Merchant members and
// return 403 for everyone else. They are implemented for spec parity; a
// non-FCM account will see an *APIError rather than a parse failure.

// GetFCMOrders — Get FCM Orders
//
// GET /trade-api/v2/fcm/orders
//
// Orders belonging to an FCM's subtraders.
func (c *Client) GetFCMOrders(ctx context.Context, params GetFCMOrdersParams) (GetOrdersResponse, error) {
	return getJSON[GetOrdersResponse](c, ctx, pathFCM+"/orders", params.toMap())
}

// GetFCMOrdersParams are the query parameters for GetFCMOrders.
type GetFCMOrdersParams struct {
	SubtraderID    string
	ClientOrderIDs string
	EventTicker    string
	Ticker         string
	Status         string
	MinTS          int64
	MaxTS          int64
	Limit          int
	Cursor         string
}

func (p GetFCMOrdersParams) toMap() map[string]string {
	return NewQuery().
		String("subtrader_id", p.SubtraderID).
		String("client_order_ids", p.ClientOrderIDs).
		String("event_ticker", p.EventTicker).
		String("ticker", p.Ticker).
		String("status", p.Status).
		Int64("min_ts", p.MinTS).
		Int64("max_ts", p.MaxTS).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}

// GetFCMPositions — Get FCM Positions
//
// GET /trade-api/v2/fcm/positions
//
// Positions for a single FCM subtrader. SubtraderID is required.
func (c *Client) GetFCMPositions(ctx context.Context, params GetFCMPositionsParams) (GetPositionsResponse, error) {
	return getJSON[GetPositionsResponse](c, ctx, pathFCM+"/positions", params.toMap())
}

// GetFCMPositionsParams are the query parameters for GetFCMPositions.
type GetFCMPositionsParams struct {
	// SubtraderID is required by the API.
	SubtraderID      string
	Ticker           string
	EventTicker      string
	CountFilter      string
	SettlementStatus string
	Limit            int
	Cursor           string
}

func (p GetFCMPositionsParams) toMap() map[string]string {
	return NewQuery().
		String("subtrader_id", p.SubtraderID).
		String("ticker", p.Ticker).
		String("event_ticker", p.EventTicker).
		String("count_filter", p.CountFilter).
		String("settlement_status", p.SettlementStatus).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}
