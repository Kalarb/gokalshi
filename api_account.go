package gokalshi

import (
	"context"
)

const pathAccount = "/trade-api/v2/account"

// GetAccountAPILimits retrieves the API tier limits for the authenticated user.
func (c *Client) GetAccountAPILimits(ctx context.Context) (GetAccountAPILimitsResponse, error) {
	return getJSON[GetAccountAPILimitsResponse](c, ctx, pathAccount+"/limits", nil)
}
