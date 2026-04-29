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
