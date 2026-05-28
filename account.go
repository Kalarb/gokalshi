package gokalshi

import (
	"context"
)

// GetAccountAPILimits — Get Account API Limits
//
// GET /trade-api/v2/account/limits
//
// Endpoint to retrieve the API tier limits associated with the authenticated
// user.
//
// See https://trading-api.readme.io/reference/getaccountapilimits
func (c *Client) GetAccountAPILimits(ctx context.Context) (GetAccountApiLimitsResponse, error) {
	return getJSON[GetAccountApiLimitsResponse](c, ctx, pathAccount+"/limits", nil)
}

// GetAccountEndpointCosts — List Non-Default Endpoint Costs
//
// GET /trade-api/v2/account/endpoint_costs
//
// Lists API v2 endpoints whose configured token cost differs from the default
// cost. Endpoints that use the default cost are omitted.
//
// See https://trading-api.readme.io/reference/getaccountendpointcosts
func (c *Client) GetAccountEndpointCosts(ctx context.Context) (GetAccountEndpointCostsResponse, error) {
	return getJSON[GetAccountEndpointCostsResponse](c, ctx, pathAccount+"/endpoint_costs", nil)
}
