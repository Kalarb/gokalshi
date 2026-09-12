package gokalshi

import (
	"context"
	"fmt"
)

// IntraExchangeInstanceTransfer — Intra Exchange Instance Transfer
//
// POST /trade-api/v2/portfolio/intra_exchange_instance_transfer
//
// Moves balance between exchange instances — event-contract and margined — for
// the authenticated member, optionally scoped to a subaccount.
func (c *Client) IntraExchangeInstanceTransfer(ctx context.Context, req IntraExchangeInstanceTransferRequest) (IntraExchangeInstanceTransferResponse, error) {
	return postJSON[IntraExchangeInstanceTransferResponse](c, ctx, pathIntraExchangeTransfer, req, 10.0)
}

// GetIntraExchangeInstanceTransfers — Get Intra Exchange Instance Transfers
//
// GET /trade-api/v2/portfolio/intra_exchange_instance_transfers
//
// Transfers are asynchronous: a transfer can be returned as pending before it
// completes.
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

// GetIntraExchangeInstanceTransfer — Get Intra Exchange Instance Transfer
//
// GET /trade-api/v2/portfolio/intra_exchange_instance_transfers/{transfer_id}
//
// Looks up a single transfer, which is how you observe a pending transfer
// reaching complete.
func (c *Client) GetIntraExchangeInstanceTransfer(ctx context.Context, transferID string) (GetIntraExchangeInstanceTransferResponse, error) {
	path := fmt.Sprintf("%s/%s", pathIntraExchangeTransfers, transferID)
	return getJSON[GetIntraExchangeInstanceTransferResponse](c, ctx, path, nil)
}
