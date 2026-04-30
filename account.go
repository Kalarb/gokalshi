package gokalshi

import (
	"context"
)

const pathAccount = "/trade-api/v2/account"

// GetAccountAPILimits — Get Account API Limits
//
// GET /trade-api/v2/account/limits
//
// Endpoint to retrieve the API tier limits associated with the authenticated
// user.
//
// See https://trading-api.readme.io/reference/getaccountapilimits
func (c *Client) GetAccountAPILimits(ctx context.Context) (GetAccountAPILimitsResponse, error) {
	return getJSON[GetAccountAPILimitsResponse](c, ctx, pathAccount+"/limits", nil)
}

// GetAccountAPILimitsResponse is the response from GET /account/limits.
type GetAccountAPILimitsResponse struct {
	UsageTier  string `json:"usage_tier"`
	ReadLimit  int    `json:"read_limit"`
	WriteLimit int    `json:"write_limit"`
}
