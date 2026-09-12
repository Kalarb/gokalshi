package gokalshi

import "context"

// GetTargetBalanceAllocation — Get Target Balance Allocation
//
// GET /trade-api/v2/portfolio/target_balance_allocation
//
// The account's target split of balance across exchange instances, and the
// policy governing how much collateral an automatic rebalance leaves behind
// for resting orders.
func (c *Client) GetTargetBalanceAllocation(ctx context.Context) (GetTargetBalanceAllocationResponse, error) {
	return getJSON[GetTargetBalanceAllocationResponse](c, ctx, pathTargetBalanceAllocation, nil)
}

// SetTargetBalanceAllocation — Set Target Balance Allocation
//
// POST /trade-api/v2/portfolio/target_balance_allocation
//
// Sets the target allocation. Kalshi rebalances toward it automatically, so
// this changes standing behaviour rather than moving funds once.
func (c *Client) SetTargetBalanceAllocation(ctx context.Context, req SetTargetBalanceAllocationRequest) (EmptyResponse, error) {
	return postJSON[EmptyResponse](c, ctx, pathTargetBalanceAllocation, req, 10.0)
}
