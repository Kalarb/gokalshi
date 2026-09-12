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

// UpgradeAPIUsageLevel — Upgrade Account API Usage Level
//
// POST /trade-api/v2/account/api_usage_level/upgrade
//
// Grants a permanent Advanced API usage-level grant. Criteria: at least 1 of
// the user's last 100 orders was created via API. Use GetAccountAPILimits to
// inspect the resulting usage tier and grants.
//
// See https://trading-api.readme.io/reference/upgradeaccountapiusagelevel
func (c *Client) UpgradeAPIUsageLevel(ctx context.Context) error {
	_, err := c.post(ctx, pathAccount+"/api_usage_level/upgrade", nil, 30.0)
	return err
}

// GetAccountAPIUsageLevelVolumeProgress — Get Account API Usage Level Volume Progress
//
// GET /trade-api/v2/account/api_usage_level/volume_progress
//
// Reports progress toward the trading-volume goals that promote an account to
// the next API usage level, which is what raises its rate-limit budget.
func (c *Client) GetAccountAPIUsageLevelVolumeProgress(ctx context.Context) (GetAccountApiUsageLevelVolumeProgressResponse, error) {
	return getJSON[GetAccountApiUsageLevelVolumeProgressResponse](c, ctx, pathAccount+"/api_usage_level/volume_progress", nil)
}
