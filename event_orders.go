package gokalshi

import (
	"context"
	"fmt"
)

// CreateOrderV2 â€” Create Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders
//
// Endpoint for submitting event-market orders using the V2 request/response
// shape (single-book `bid`/`ask` side and fixed-point dollar prices). The
// legacy `/portfolio/orders` endpoint will be deprecated no earlier than May
// 6, 2026 â€” clients should migrate to this path.
//
// See https://docs.kalshi.com/api-reference/orders/create-order-v2
func (c *Client) CreateOrderV2(ctx context.Context, req CreateOrderV2Request) (CreateOrderV2Response, error) {
	return postJSON[CreateOrderV2Response](c, ctx, pathEventOrders, req, 10.0)
}

// BatchCreateOrdersV2 â€” Batch Create Orders (V2)
//
// POST /trade-api/v2/portfolio/events/orders/batched
//
// Endpoint for submitting a batch of event-market orders using the V2
// request/response shape. The maximum batch size scales with your tier's write
// budget â€” see [Rate Limits and Tiers](/getting_started/rate_limits).
//
// See https://docs.kalshi.com/api-reference/orders/batch-create-orders-v2
func (c *Client) BatchCreateOrdersV2(ctx context.Context, req BatchCreateOrdersV2Request) (BatchCreateOrdersV2Response, error) {
	return postJSON[BatchCreateOrdersV2Response](c, ctx, pathEventOrders+"/batched", req, float64(len(req.Orders))*10.0)
}

// BatchCancelOrdersV2 â€” Batch Cancel Orders (V2)
//
// DELETE /trade-api/v2/portfolio/events/orders/batched
//
// Endpoint for cancelling a batch of event-market orders using the V2 response
// shape. To auto-route a cancellation, provide its `market_ticker` and omit
// `exchange_index` or set it to `-1`. The maximum batch size scales with your
// tier's write budget â€” see [Rate Limits and
// Tiers](/getting_started/rate_limits).
//
// Each cancellation costs 2 write tokens.
//
// See https://docs.kalshi.com/api-reference/orders/batch-cancel-orders-v2
func (c *Client) BatchCancelOrdersV2(ctx context.Context, req BatchCancelOrdersV2Request) (BatchCancelOrdersV2Response, error) {
	if len(req.Orders) == 0 {
		return BatchCancelOrdersV2Response{}, fmt.Errorf("batch cancellation requires at least one order")
	}
	// The SDK owns the limiter and therefore the batch size. Each request must
	// fit the bucket even when account-limit discovery fell back to Basic.
	c.mu.RLock()
	_, cost := c.resolveCosts("DELETE", pathEventOrders+"/batched", 0, 2)
	capacity := c.limiter.cfg.writeCap()
	c.mu.RUnlock()
	if cost <= 0 || capacity < cost {
		return BatchCancelOrdersV2Response{}, fmt.Errorf("cancellation cost %g exceeds write capacity %g", cost, capacity)
	}
	size := len(req.Orders)
	if float64(size)*cost > capacity {
		size = int(capacity / cost)
	}
	var result BatchCancelOrdersV2Response
	for start := 0; start < len(req.Orders); start += size {
		end := min(start+size, len(req.Orders))
		part, err := deleteJSON[BatchCancelOrdersV2Response](c, ctx, pathEventOrders+"/batched", BatchCancelOrdersV2Request{Orders: req.Orders[start:end]}, 2)
		result.Orders = append(result.Orders, part.Orders...)
		if err != nil {
			// Earlier chunks may already have succeeded. Return their results
			// and stop; never replay a completed or uncertain cancellation.
			return result, fmt.Errorf("batch cancellation orders[%d:%d]: %w", start, end, err)
		}
	}
	return result, nil
}

// CancelOrderV2 â€” Cancel Order (V2)
//
// DELETE /trade-api/v2/portfolio/events/orders/{order_id}
//
// Endpoint for cancelling event-market orders using the V2 response shape. To
// auto-route the cancellation, provide `market_ticker` and omit
// `exchange_index` or set it to `-1`. Returns `{order_id, client_order_id,
// reduced_by}` rather than a full order object.
//
// See https://docs.kalshi.com/api-reference/orders/cancel-order-v2
func (c *Client) CancelOrderV2(ctx context.Context, orderID string, params CancelOrderV2Params) (CancelOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s", pathEventOrders, orderID)
	return doJSON[CancelOrderV2Response](c, ctx, "DELETE", path, 0, 2.0, nil, params.toMap())
}

// AmendOrderV2 â€” Amend Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders/{order_id}/amend
//
// Endpoint for amending the price and/or max fillable count of an existing
// event-market order using the V2 request/response shape. The request `count`
// is the updated total/max fillable count, equal to already filled count plus
// desired resting remaining count. This behavior matches the v1 amend
// endpoints; only the request/response shape differs.
//
// See https://docs.kalshi.com/api-reference/orders/amend-order-v2
func (c *Client) AmendOrderV2(ctx context.Context, orderID string, req AmendOrderV2Request) (AmendOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s/amend", pathEventOrders, orderID)
	return postJSON[AmendOrderV2Response](c, ctx, path, req, 10.0)
}

// DecreaseOrderV2 â€” Decrease Order (V2)
//
// POST /trade-api/v2/portfolio/events/orders/{order_id}/decrease
//
// Endpoint for decreasing the remaining count of an existing event-market
// order using the V2 request/response shape. Exactly one of `reduce_by` or
// `reduce_to` must be provided.
//
// See https://docs.kalshi.com/api-reference/orders/decrease-order-v2
func (c *Client) DecreaseOrderV2(ctx context.Context, orderID string, req DecreaseOrderV2Request) (DecreaseOrderV2Response, error) {
	path := fmt.Sprintf("%s/%s/decrease", pathEventOrders, orderID)
	return postJSON[DecreaseOrderV2Response](c, ctx, path, req, 10.0)
}

// CancelOrderV2Params are query parameters for CancelOrderV2.
type CancelOrderV2Params struct {
	Subaccount int
	// MarketTicker lets Kalshi route the cancellation when ExchangeIndex is omitted.
	MarketTicker string
	// ExchangeIndex selects the exchange instance. 0 is the event-contract
	// instance and -1 asks Kalshi to auto-route by market ticker, so it is a
	// pointer: both are meaningful values that a zero-valued int cannot express.
	ExchangeIndex *int
}

func (p CancelOrderV2Params) toMap() map[string]string {
	return NewQuery().
		Int("subaccount", p.Subaccount).
		IntPtr("exchange_index", p.ExchangeIndex).
		String("market_ticker", p.MarketTicker).
		Build()
}

// CancelAllOrders â€” Cancel All Orders
//
// DELETE /trade-api/v2/portfolio/events/orders
//
// Cancels all resting event-market orders for the authenticated Direct member
// across every exchange shard. If `subaccount` is omitted, matching orders may
// come from any subaccount. If it is provided, only orders for that subaccount
// are eligible. Newly placed orders may also be cancelled during the minute
// after the request.
//
// See https://docs.kalshi.com/api-reference/orders/cancel-all-orders
func (c *Client) CancelAllOrders(ctx context.Context, params CancelAllOrdersParams) error {
	_, err := c.do(ctx, "DELETE", pathEventOrders, 0, 2.0, nil, params.toMap())
	return err
}

// CancelAllOrdersParams are the query parameters for CancelAllOrders.
type CancelAllOrdersParams struct {
	Subaccount int
}

func (p CancelAllOrdersParams) toMap() map[string]string {
	return NewQuery().
		Int("subaccount", p.Subaccount).
		Build()
}
