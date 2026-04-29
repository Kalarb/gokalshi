package gokalshi

import (
	"context"
	"fmt"
)

const (
	pathPortfolio = "/trade-api/v2/portfolio"
	pathOrders    = pathPortfolio + "/orders"
)

// CreateOrder submits a new order.
func (c *Client) CreateOrder(ctx context.Context, req CreateOrderRequest) (SingleCreateResponse, error) {
	return postJSON[SingleCreateResponse](c, ctx, pathOrders, req, 1.0)
}

// CancelOrder cancels an existing order by ID.
func (c *Client) CancelOrder(ctx context.Context, orderID string) (CancelOrderResponse, error) {
	path := fmt.Sprintf("%s/%s", pathOrders, orderID)
	return deleteJSON[CancelOrderResponse](c, ctx, path, nil, 0.2)
}

// GetOrder retrieves a single order by ID.
func (c *Client) GetOrder(ctx context.Context, orderID string) (SingleCreateResponse, error) {
	path := fmt.Sprintf("%s/%s", pathOrders, orderID)
	return getJSON[SingleCreateResponse](c, ctx, path, nil)
}

// GetOrders retrieves orders matching the given parameters.
func (c *Client) GetOrders(ctx context.Context, params GetOrdersParams) (GetOrdersResponse, error) {
	return getJSON[GetOrdersResponse](c, ctx, pathOrders, params.toMap())
}

// BatchCreateOrders submits multiple orders in a single API call (max 20).
func (c *Client) BatchCreateOrders(ctx context.Context, orders []CreateOrderRequest) (BatchCreateResponse, error) {
	body := BatchCreateRequest{Orders: orders}
	cost := float64(len(orders))
	return postJSON[BatchCreateResponse](c, ctx, pathOrders+"/batched", body, cost)
}

// BatchCancelOrders cancels multiple orders in a single API call (max 20).
func (c *Client) BatchCancelOrders(ctx context.Context, orders []BatchCancelOrderEntry) (BatchCancelResponse, error) {
	body := BatchCancelRequest{Orders: orders}
	cost := float64(len(orders)) * 0.2
	return deleteJSON[BatchCancelResponse](c, ctx, pathOrders+"/batched", body, cost)
}

// AmendOrder modifies an existing order's price and/or quantity.
func (c *Client) AmendOrder(ctx context.Context, orderID string, req AmendOrderRequest) (AmendOrderResponse, error) {
	path := fmt.Sprintf("%s/%s/amend", pathOrders, orderID)
	return postJSON[AmendOrderResponse](c, ctx, path, req, 1.0)
}

// DecreaseOrder reduces the size of an existing order.
func (c *Client) DecreaseOrder(ctx context.Context, orderID string, req DecreaseOrderRequest) (SingleCreateResponse, error) {
	path := fmt.Sprintf("%s/%s/decrease", pathOrders, orderID)
	return postJSON[SingleCreateResponse](c, ctx, path, req, 1.0)
}

// GetQueuePositions retrieves queue positions for matching orders.
func (c *Client) GetQueuePositions(ctx context.Context, params GetQueuePositionsParams) (GetQueuePositionsResponse, error) {
	return getJSON[GetQueuePositionsResponse](c, ctx, pathOrders+"/queue_positions", params.toMap())
}

// GetQueuePosition retrieves the queue position for a single order.
func (c *Client) GetQueuePosition(ctx context.Context, orderID string) (GetQueuePositionResponse, error) {
	path := fmt.Sprintf("%s/%s/queue_position", pathOrders, orderID)
	return getJSON[GetQueuePositionResponse](c, ctx, path, nil)
}
