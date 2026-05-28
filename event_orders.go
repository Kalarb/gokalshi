package gokalshi

import (
	"context"
	"fmt"
)

const pathEventOrders = pathPortfolio + "/events/orders"

// CreateOrderV2 — Create Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders
//
// Endpoint for submitting event-market orders using the V2 request/response
// shape (single-book `bid`/`ask` side and fixed-point dollar prices). The
// legacy `/portfolio/orders` endpoint will be deprecated no earlier than May
// 6, 2026 — clients should migrate to this path.
//
// See https://trading-api.readme.io/reference/createorderv2
func (c *Client) CreateOrderV2(ctx context.Context, req CreateOrderV2Request) (CreateOrderV2Response, error) {
	return postJSON[CreateOrderV2Response](c, ctx, pathEventOrders, req, 1.0)
}

// BatchCreateOrdersV2 — Batch Create Orders (V2)
//
// POST /trade-api/v2/portfolio/events/orders/batched
//
// Endpoint for submitting a batch of event-market orders using the V2
// request/response shape. The maximum batch size scales with your tier's write
// budget — see [Rate Limits and Tiers](/getting_started/rate_limits).
//
// See https://trading-api.readme.io/reference/batchcreateordersv2
func (c *Client) BatchCreateOrdersV2(ctx context.Context, req BatchCreateOrdersV2Request) (BatchCreateOrdersV2Response, error) {
	return postJSON[BatchCreateOrdersV2Response](c, ctx, pathEventOrders+"/batched", req, float64(len(req.Orders)))
}

// BatchCancelOrdersV2 — Batch Cancel Orders (V2)
//
// DELETE /trade-api/v2/portfolio/events/orders/batched
//
// Endpoint for cancelling a batch of event-market orders using the V2 response
// shape. The maximum batch size scales with your tier's write budget — see
// [Rate Limits and Tiers](/getting_started/rate_limits).
//
// See https://trading-api.readme.io/reference/batchcancelordersv2
func (c *Client) BatchCancelOrdersV2(ctx context.Context, req BatchCancelOrdersV2Request) (BatchCancelOrdersV2Response, error) {
	return deleteJSON[BatchCancelOrdersV2Response](c, ctx, pathEventOrders+"/batched", req, float64(len(req.Orders)))
}

// CancelOrderV2 — Cancel Order (V2)
//
// DELETE /trade-api/v2/portfolio/events/orders/{order_id}
//
// Endpoint for cancelling event-market orders using the V2 response shape.
// Returns `{order_id, client_order_id, reduced_by}` rather than a full order
// object.
//
// See https://trading-api.readme.io/reference/cancelorderv2
func (c *Client) CancelOrderV2(ctx context.Context, orderID string, params CancelOrderV2Params) (CancelOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s", pathEventOrders, orderID)
	return doJSON[CancelOrderV2Response](c, ctx, "DELETE", path, 0, 1.0, nil, params.toMap())
}

// AmendOrderV2 — Amend Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders/{order_id}/amend
//
// Endpoint for amending the price and/or max fillable count of an existing
// event-market order using the V2 request/response shape. The request `count`
// is the updated total/max fillable count, equal to already filled count plus
// desired resting remaining count. This behavior matches the v1 amend
// endpoints; only the request/response shape differs.
//
// See https://trading-api.readme.io/reference/amendorderv2
func (c *Client) AmendOrderV2(ctx context.Context, orderID string, req AmendOrderV2Request) (AmendOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s/amend", pathEventOrders, orderID)
	return postJSON[AmendOrderV2Response](c, ctx, path, req, 1.0)
}

// DecreaseOrderV2 — Decrease Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders/{order_id}/decrease
//
// Endpoint for decreasing the remaining count of an existing event-market
// order using the V2 request/response shape. Exactly one of `reduce_by` or
// `reduce_to` must be provided.
//
// See https://trading-api.readme.io/reference/decreaseorderv2
func (c *Client) DecreaseOrderV2(ctx context.Context, orderID string, req DecreaseOrderV2Request) (DecreaseOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s/decrease", pathEventOrders, orderID)
	return postJSON[DecreaseOrderV2Response](c, ctx, path, req, 1.0)
}

// CancelOrderV2Params are query parameters for CancelOrderV2.
type CancelOrderV2Params struct {
	Subaccount    int
	ExchangeIndex int
}

func (p CancelOrderV2Params) toMap() map[string]string {
	return NewQuery().
		Int("subaccount", p.Subaccount).
		Int("exchange_index", p.ExchangeIndex).
		Build()
}
