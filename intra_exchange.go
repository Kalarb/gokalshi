package gokalshi

import (
	"context"
	"fmt"
)

// IntraExchangeInstanceTransfer — Intra Account Transfer
//
// POST /trade-api/v2/portfolio/intra_exchange_instance_transfer
//
// Transfers funds within the same account.
//
// See https://docs.kalshi.com/api-reference/portfolio/intra-exchange-instance-transfer
func (c *Client) IntraExchangeInstanceTransfer(ctx context.Context, req IntraExchangeInstanceTransferRequest) (IntraExchangeInstanceTransferResponse, error) {
	return postJSON[IntraExchangeInstanceTransferResponse](c, ctx, pathIntraExchangeTransfer, req, 10.0)
}

// GetIntraExchangeInstanceTransfers — Get Intra Account Transfers
//
// GET /trade-api/v2/portfolio/intra_exchange_instance_transfers
//
// Endpoint for fetching intra-exchange account transfer history.
//
// See https://docs.kalshi.com/api-reference/portfolio/get-intra-exchange-instance-transfers
func (c *Client) GetIntraExchangeInstanceTransfers(ctx context.Context, params GetIntraExchangeInstanceTransfersParams) (GetIntraExchangeInstanceTransfersResponse, error) {
	return getJSON[GetIntraExchangeInstanceTransfersResponse](c, ctx, pathIntraExchangeTransfers, params.toMap())
}

// GetIntraExchangeInstanceTransfersParams are the query parameters for
// GetIntraExchangeInstanceTransfers.
type GetIntraExchangeInstanceTransfersParams struct {
	Limit  int
	Cursor string
}

func (p GetIntraExchangeInstanceTransfersParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}

// GetIntraExchangeInstanceTransfer — Get Intra Account Transfer
//
// GET /trade-api/v2/portfolio/intra_exchange_instance_transfers/{transfer_id}
//
// Endpoint for getting a single intra-account transfer by id.
//
// See https://docs.kalshi.com/api-reference/portfolio/get-intra-exchange-instance-transfer
func (c *Client) GetIntraExchangeInstanceTransfer(ctx context.Context, transferID string) (GetIntraExchangeInstanceTransferResponse, error) {
	path := fmt.Sprintf("%s/%s", pathIntraExchangeTransfers, transferID)
	return getJSON[GetIntraExchangeInstanceTransferResponse](c, ctx, path, nil)
}
