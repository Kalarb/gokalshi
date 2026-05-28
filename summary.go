package gokalshi

import "context"

// GetPortfolioRestingOrderTotalValue — Get Total Resting Order Value
//
// GET /trade-api/v2/portfolio/summary/total_resting_order_value
//
// Endpoint for getting the total value, in cents, of resting orders. This
// endpoint is only intended for use by FCM members (rare). Note: If you're
// uncertain about this endpoint, it likely does not apply to you.
//
// See https://trading-api.readme.io/reference/getportfoliorestingordertotalvalue
func (c *Client) GetPortfolioRestingOrderTotalValue(ctx context.Context) (GetPortfolioRestingOrderTotalValueResponse, error) {
	return getJSON[GetPortfolioRestingOrderTotalValueResponse](c, ctx, pathPortfolio+"/summary/total_resting_order_value", nil)
}
