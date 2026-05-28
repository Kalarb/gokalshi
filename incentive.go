package gokalshi

import "context"

// GetIncentivePrograms — Get Incentives
//
// GET /trade-api/v2/incentive_programs
//
// List incentives with optional filters. Incentives are rewards programs for
// trading activity on specific markets.
//
// See https://trading-api.readme.io/reference/getincentiveprograms
func (c *Client) GetIncentivePrograms(ctx context.Context) (GetIncentiveProgramsResponse, error) {
	return getJSON[GetIncentiveProgramsResponse](c, ctx, "/trade-api/v2/incentive_programs", nil)
}
