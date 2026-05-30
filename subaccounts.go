package gokalshi

import "context"

// CreateSubaccount — Create Subaccount
//
// POST /trade-api/v2/portfolio/subaccounts
//
// Creates a new subaccount for the authenticated user. This endpoint is
// currently only available to institutions and market makers. Subaccounts are
// numbered sequentially starting from 1. Maximum 32 subaccounts per user.
//
// See https://trading-api.readme.io/reference/createsubaccount
func (c *Client) CreateSubaccount(ctx context.Context) (CreateSubaccountResponse, error) {
	return postJSON[CreateSubaccountResponse](c, ctx, pathSubaccounts, nil, 10.0)
}

// GetSubaccountBalances — Get All Subaccount Balances
//
// GET /trade-api/v2/portfolio/subaccounts/balances
//
// Gets balances for all subaccounts including the primary account.
//
// See https://trading-api.readme.io/reference/getsubaccountbalances
func (c *Client) GetSubaccountBalances(ctx context.Context) (GetSubaccountBalancesResponse, error) {
	return getJSON[GetSubaccountBalancesResponse](c, ctx, pathSubaccounts+"/balances", nil)
}

// GetSubaccountNetting — Get Subaccount Netting
//
// GET /trade-api/v2/portfolio/subaccounts/netting
//
// Gets the netting enabled settings for all subaccounts.
//
// See https://trading-api.readme.io/reference/getsubaccountnetting
func (c *Client) GetSubaccountNetting(ctx context.Context) (GetSubaccountNettingResponse, error) {
	return getJSON[GetSubaccountNettingResponse](c, ctx, pathSubaccounts+"/netting", nil)
}

// UpdateSubaccountNetting — Update Subaccount Netting
//
// PUT /trade-api/v2/portfolio/subaccounts/netting
//
// Updates the netting enabled setting for a specific subaccount. Use 0 for the
// primary account, or 1-32 for numbered subaccounts.
//
// See https://trading-api.readme.io/reference/updatesubaccountnetting
func (c *Client) UpdateSubaccountNetting(ctx context.Context, req UpdateSubaccountNettingRequest) error {
	_, err := c.put(ctx, pathSubaccounts+"/netting", req, 10.0)
	return err
}

// ApplySubaccountTransfer — Transfer Between Subaccounts
//
// POST /trade-api/v2/portfolio/subaccounts/transfer
//
// Transfers funds between the authenticated user's subaccounts. Use 0 for the
// primary account, or 1-32 for numbered subaccounts.
//
// See https://trading-api.readme.io/reference/applysubaccounttransfer
func (c *Client) ApplySubaccountTransfer(ctx context.Context, req ApplySubaccountTransferRequest) (ApplySubaccountTransferResponse, error) {
	return postJSON[ApplySubaccountTransferResponse](c, ctx, pathSubaccounts+"/transfer", req, 10.0)
}

// GetSubaccountTransfers — Get Subaccount Transfers
//
// GET /trade-api/v2/portfolio/subaccounts/transfers
//
// Gets a paginated list of all transfers between subaccounts for the
// authenticated user.
//
// See https://trading-api.readme.io/reference/getsubaccounttransfers
func (c *Client) GetSubaccountTransfers(ctx context.Context, params GetSubaccountTransfersParams) (GetSubaccountTransfersResponse, error) {
	return getJSON[GetSubaccountTransfersResponse](c, ctx, pathSubaccounts+"/transfers", params.toMap())
}

// GetSubaccountTransfersParams are query parameters for GetSubaccountTransfers.
type GetSubaccountTransfersParams struct {
	Limit  int
	Cursor string
}

func (p GetSubaccountTransfersParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}
