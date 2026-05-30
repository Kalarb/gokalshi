package gokalshi

import (
	"context"
	"fmt"
)

// CreateOrder — Create Order
//
// POST /trade-api/v2/portfolio/orders
//
// Endpoint for submitting orders in a market. Each user is limited to 200 000
// open orders at a time.
//
// See https://trading-api.readme.io/reference/createorder
func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (CreateOrderResponse, error) {
	return postJSON[CreateOrderResponse](c, ctx, pathOrders, req, 10.0)
}

// CancelOrder — Cancel Order
//
// DELETE /trade-api/v2/portfolio/orders/{order_id}
//
// Endpoint for canceling orders. The value for the orderId should match the id
// field of the order you want to decrease. Commonly, DELETE-type endpoints
// return 204 status with no body content on success. But we can't completely
// delete the order, as it may be partially filled already. Instead, the
// DeleteOrder endpoint reduce the order completely, essentially zeroing the
// remaining resting contracts on it. The zeroed order is returned on the
// response payload as a form of validation for the client.
//
// See https://trading-api.readme.io/reference/cancelorder
func (c *Client) CancelOrder(ctx context.Context, orderID string) (CancelOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", pathOrders, orderID)
	return deleteJSON[CancelOrderResponse](c, ctx, path, nil, 10.0)
}

// GetOrder — Get Order
//
// GET /trade-api/v2/portfolio/orders/{order_id}
//
// Endpoint for getting a single order.
//
// See https://trading-api.readme.io/reference/getorder
func (c *Client) GetOrder(ctx context.Context, orderID string) (CreateOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", pathOrders, orderID)
	return getJSON[CreateOrderResponse](c, ctx, path, nil)
}

// GetOrders — Get Orders
//
// GET /trade-api/v2/portfolio/orders
//
// Restricts the response to orders that have a certain status: resting,
// canceled, or executed. Orders that have been canceled or fully executed
// before the historical cutoff are only available via `GET
// /historical/orders`. Resting orders will always be available through this
// endpoint. See [Historical
// Data](https://docs.kalshi.com/getting_started/historical_data) for details.
//
// See https://trading-api.readme.io/reference/getorders
func (c *Client) GetOrders(ctx context.Context, params GetOrdersParams) (GetOrdersResponse, error) {
	return getJSON[GetOrdersResponse](c, ctx, pathOrders, params.toMap())
}

// BatchCreateOrders — Batch Create Orders
//
// POST /trade-api/v2/portfolio/orders/batched
//
// Endpoint for submitting a batch of orders. The maximum batch size scales
// with your tier's write budget — see [Rate Limits and
// Tiers](/getting_started/rate_limits).
//
// See https://trading-api.readme.io/reference/batchcreateorders
func (c *Client) BatchCreateOrders(ctx context.Context, orders []CreateOrderRequest) (BatchCreateOrdersResponse, error) {
	body := BatchCreateOrdersRequest{Orders: orders}
	cost := float64(len(orders)) * 10.0
	return postJSON[BatchCreateOrdersResponse](c, ctx, pathOrders+"/batched", body, cost)
}

// BatchCancelOrders — Batch Cancel Orders
//
// DELETE /trade-api/v2/portfolio/orders/batched
//
// Endpoint for cancelling a batch of orders. The maximum batch size scales
// with your tier's write budget — see [Rate Limits and
// Tiers](/getting_started/rate_limits).
//
// See https://trading-api.readme.io/reference/batchcancelorders
func (c *Client) BatchCancelOrders(ctx context.Context, orders []BatchCancelOrdersRequestOrder) (BatchCancelOrdersResponse, error) {
	body := BatchCancelOrdersRequest{Orders: orders}
	cost := float64(len(orders)) * 10.0
	return deleteJSON[BatchCancelOrdersResponse](c, ctx, pathOrders+"/batched", body, cost)
}

// AmendOrder — Amend Order
//
// POST /trade-api/v2/portfolio/orders/{order_id}/amend
//
// Endpoint for amending the max number of fillable contracts and/or price in
// an existing order. Max fillable contracts is `remaining_count` +
// `fill_count`.
//
// See https://trading-api.readme.io/reference/amendorder
func (c *Client) AmendOrder(ctx context.Context, orderID string, req AmendOrderRequest) (AmendOrderResponse, error) {
	path := fmt.Sprintf("%s/%s/amend", pathOrders, orderID)
	return postJSON[AmendOrderResponse](c, ctx, path, req, 10.0)
}

// DecreaseOrder — Decrease Order
//
// POST /trade-api/v2/portfolio/orders/{order_id}/decrease
//
// Endpoint for decreasing the number of contracts in an existing order. This
// is the only kind of edit available on order quantity. Cancelling an order is
// equivalent to decreasing an order amount to zero.
//
// See https://trading-api.readme.io/reference/decreaseorder
func (c *Client) DecreaseOrder(ctx context.Context, orderID string, req DecreaseOrderRequest) (CreateOrderResponse, error) {
	path := fmt.Sprintf("%s/%s/decrease", pathOrders, orderID)
	return postJSON[CreateOrderResponse](c, ctx, path, req, 10.0)
}

// GetQueuePositions — Get Queue Positions for Orders
//
// GET /trade-api/v2/portfolio/orders/queue_positions
//
// Endpoint for getting queue positions for all resting orders. Queue position
// represents the number of contracts that need to be matched before an order
// receives a partial or full match, determined using price-time priority.
//
// See https://trading-api.readme.io/reference/getorderqueuepositions
func (c *Client) GetQueuePositions(ctx context.Context, params GetQueuePositionsParams) (GetOrderQueuePositionsResponse, error) {
	return getJSON[GetOrderQueuePositionsResponse](c, ctx, pathOrders+"/queue_positions", params.toMap())
}

// GetQueuePosition — Get Order Queue Position
//
// GET /trade-api/v2/portfolio/orders/{order_id}/queue_position
//
// Endpoint for getting an order's queue position in the order book. This
// represents the amount of orders that need to be matched before this order
// receives a partial or full match. Queue position is determined using a
// price-time priority.
//
// See https://trading-api.readme.io/reference/getorderqueueposition
func (c *Client) GetQueuePosition(ctx context.Context, orderID string) (GetOrderQueuePositionResponse, error) {
	path := fmt.Sprintf("%s/%s/queue_position", pathOrders, orderID)
	return getJSON[GetOrderQueuePositionResponse](c, ctx, path, nil)
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetOrdersParams holds optional query parameters for GetOrders.
type GetOrdersParams struct {
	Ticker      string
	EventTicker string
	Status      OrderStatus
	Limit       int
	Cursor      string
	MinTs       int64
	MaxTs       int64
	Subaccount  int
}

func (p GetOrdersParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		String("event_ticker", p.EventTicker).
		String("status", string(p.Status)).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Int("subaccount", p.Subaccount).
		Build()
}

// GetQueuePositionsParams holds optional query parameters for GetQueuePositions.
type GetQueuePositionsParams struct {
	MarketTickers string
	EventTicker   string
	Subaccount    int
}

func (p GetQueuePositionsParams) toMap() map[string]string {
	return NewQuery().
		String("market_tickers", p.MarketTickers).
		String("event_ticker", p.EventTicker).
		Int("subaccount", p.Subaccount).
		Build()
}
