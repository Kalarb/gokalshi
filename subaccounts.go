package gokalshi

import "context"

// CreateSubaccount — Create Subaccount
//
// POST /trade-api/v2/portfolio/subaccounts
//
// Creates a new subaccount for the authenticated user. This endpoint is
// available to all users on the Advanced API tier and above. Subaccounts are
// numbered sequentially starting from 1. Maximum 63 numbered subaccounts per
// user (64 including the primary account).
//
// See https://docs.kalshi.com/api-reference/portfolio/create-subaccount
func (c *Client) CreateSubaccount(ctx context.Context) (CreateSubaccountResponse, error) {
	return postJSON[CreateSubaccountResponse](c, ctx, pathSubaccounts, nil, 10.0)
}

// GetSubaccountBalances — Get All Subaccount Balances
//
// GET /trade-api/v2/portfolio/subaccounts/balances
//
// Gets balances for all subaccounts including the primary account.
//
// See https://docs.kalshi.com/api-reference/portfolio/get-subaccount-balances
func (c *Client) GetSubaccountBalances(ctx context.Context) (GetSubaccountBalancesResponse, error) {
	return getJSON[GetSubaccountBalancesResponse](c, ctx, pathSubaccounts+"/balances", nil)
}

// GetSubaccountNetting — Get Subaccount Netting
//
// GET /trade-api/v2/portfolio/subaccounts/netting
//
// Gets the netting enabled settings for all subaccounts.
//
// See https://docs.kalshi.com/api-reference/portfolio/get-subaccount-netting
func (c *Client) GetSubaccountNetting(ctx context.Context) (GetSubaccountNettingResponse, error) {
	return getJSON[GetSubaccountNettingResponse](c, ctx, pathSubaccounts+"/netting", nil)
}

// UpdateSubaccountNetting — Update Subaccount Netting
//
// PUT /trade-api/v2/portfolio/subaccounts/netting
//
// Updates the netting enabled setting for a specific subaccount. Use 0 for the
// primary account, or 1-63 for numbered subaccounts.
//
// See https://docs.kalshi.com/api-reference/portfolio/update-subaccount-netting
func (c *Client) UpdateSubaccountNetting(ctx context.Context, req UpdateSubaccountNettingRequest) error {
	_, err := c.put(ctx, pathSubaccounts+"/netting", req, 10.0)
	return err
}

// ApplySubaccountTransfer — Transfer Between Subaccounts
//
// POST /trade-api/v2/portfolio/subaccounts/transfer
//
// Transfers funds between the authenticated user's subaccounts. Use 0 for the
// primary account, or 1-63 for numbered subaccounts. Set exchange_index to
// apply the transfer on a specific exchange shard (defaults to 0).
//
// See https://docs.kalshi.com/api-reference/portfolio/apply-subaccount-transfer
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
// See https://docs.kalshi.com/api-reference/portfolio/get-subaccount-transfers
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
