package gokalshi

import "context"

// GetTargetBalanceAllocation — Get Target Balance Allocation
//
// GET /trade-api/v2/portfolio/target_balance_allocation
//
// Retrieves the caller's target balance allocation across exchange indexes.
//
// See https://docs.kalshi.com/api-reference/portfolio/get-target-balance-allocation
func (c *Client) GetTargetBalanceAllocation(ctx context.Context) (GetTargetBalanceAllocationResponse, error) {
	return getJSON[GetTargetBalanceAllocationResponse](c, ctx, pathTargetBalanceAllocation, nil)
}

// SetTargetBalanceAllocation — Set Target Balance Allocation
//
// POST /trade-api/v2/portfolio/target_balance_allocation
//
// Replaces the caller's target balance allocation across exchange indexes.
// Percentages must total 100. Passing an empty allocations array disables
// automatic rebalancing.
//
// SDK note: to disable automatic rebalancing pass an explicitly empty
// slice — SetTargetBalanceAllocationRequest{Allocations: []TargetBalanceAllocationInput{}}.
// A nil Allocations marshals to JSON null rather than [], which is a
// different request and not the documented way to disable it.
//
// See https://docs.kalshi.com/api-reference/portfolio/set-target-balance-allocation
func (c *Client) SetTargetBalanceAllocation(ctx context.Context, req SetTargetBalanceAllocationRequest) (EmptyResponse, error) {
	return postJSON[EmptyResponse](c, ctx, pathTargetBalanceAllocation, req, 10.0)
}
