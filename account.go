package gokalshi

import (
	"context"
)

// GetAccountAPILimits — Get Account API Limits
//
// GET /trade-api/v2/account/limits
//
// Endpoint to retrieve the authenticated user's Predictions API usage tier and
// token-bucket limits. Public Predictions tiers include Basic, Advanced,
// Expert, Premier, Paragon, Prime, and Prestige.
//
// See https://docs.kalshi.com/api-reference/account/get-account-api-limits
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
// See https://docs.kalshi.com/api-reference/account/get-account-endpoint-costs
func (c *Client) GetAccountEndpointCosts(ctx context.Context) (GetAccountEndpointCostsResponse, error) {
	return getJSON[GetAccountEndpointCostsResponse](c, ctx, pathAccount+"/endpoint_costs", nil)
}

// UpgradeAPIUsageLevel — Upgrade Account API Usage Level
//
// POST /trade-api/v2/account/api_usage_level/upgrade
//
// Grants a permanent Advanced API usage-level grant. Currently only the
// Predictions exchange instance is supported. Criteria: at least 1 of the
// user's last 100 Predictions orders was created via API. Use Get Account API
// Limits to inspect the resulting usage tier and grants.
//
// See https://docs.kalshi.com/api-reference/account/upgrade-account-api-usage-level
func (c *Client) UpgradeAPIUsageLevel(ctx context.Context) error {
	_, err := c.post(ctx, pathAccount+"/api_usage_level/upgrade", nil, 30.0)
	return err
}

// GetAccountAPIUsageLevelVolumeProgress — Get Account API Usage Level Volume Progress
//
// GET /trade-api/v2/account/api_usage_level/volume_progress
//
// Returns the authenticated user's latest cron-computed trading volume
// progress toward volume-based API usage tiers for the predictions
// (event_contract) lane. Volume figures are reported as fixed-point contract
// counts.
//
// See https://docs.kalshi.com/api-reference/account/get-account-api-usage-level-volume-progress
func (c *Client) GetAccountAPIUsageLevelVolumeProgress(ctx context.Context) (GetAccountApiUsageLevelVolumeProgressResponse, error) {
	return getJSON[GetAccountApiUsageLevelVolumeProgressResponse](c, ctx, pathAccount+"/api_usage_level/volume_progress", nil)
}
