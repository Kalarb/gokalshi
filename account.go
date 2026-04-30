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
func (c *Client) GetAccountAPILimits(ctx context.Context) (GetAccountApiLimitsResponse, error) {
	return getJSON[GetAccountApiLimitsResponse](c, ctx, pathAccount+"/limits", nil)
}
