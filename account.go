package gokalshi

import (
	"context"
)

const pathAccount = "/trade-api/v2/account"

// GetAccountAPILimits retrieves the API tier limits for the authenticated user.
func (c *Client) GetAccountAPILimits(ctx context.Context) (GetAccountAPILimitsResponse, error) {
	return getJSON[GetAccountAPILimitsResponse](c, ctx, pathAccount+"/limits", nil)
}

// GetAccountAPILimitsResponse is the response from GET /account/limits.
type GetAccountAPILimitsResponse struct {
	UsageTier  string `json:"usage_tier"`
	ReadLimit  int    `json:"read_limit"`
	WriteLimit int    `json:"write_limit"`
}
