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

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// CreateOrderRequest is the request body for a single order entry.
type CreateOrderRequest struct {
	Ticker                  string      `json:"ticker"`
	Side                    Side        `json:"side"`
	Action                  Action      `json:"action"`
	ClientOrderID           string      `json:"client_order_id,omitempty"`
	CountFP                 string      `json:"count_fp,omitempty"`
	YesPriceDollars         string      `json:"yes_price_dollars,omitempty"`
	NoPriceDollars          string      `json:"no_price_dollars,omitempty"`
	ExpirationTs            int64       `json:"expiration_ts,omitempty"`
	TimeInForce             TimeInForce `json:"time_in_force,omitempty"`
	BuyMaxCost              int         `json:"buy_max_cost,omitempty"`
	PostOnly                bool        `json:"post_only,omitempty"`
	ReduceOnly              bool        `json:"reduce_only,omitempty"`
	SellPositionFloor       int         `json:"sell_position_floor,omitempty"`
	SelfTradePreventionType STPType     `json:"self_trade_prevention_type,omitempty"`
	OrderGroupID            string      `json:"order_group_id,omitempty"`
	CancelOrderOnPause      bool        `json:"cancel_order_on_pause,omitempty"`
	Subaccount              int         `json:"subaccount,omitempty"`
}

// BatchCreateRequest is the request body for POST /portfolio/orders/batched.
type BatchCreateRequest struct {
	Orders []CreateOrderRequest `json:"orders"`
}

// BatchCancelOrderEntry is a single entry in a batch cancel request.
type BatchCancelOrderEntry struct {
	OrderID    string `json:"order_id"`
	Subaccount int    `json:"subaccount,omitempty"`
}

// BatchCancelRequest is the request body for DELETE /portfolio/orders/batched.
type BatchCancelRequest struct {
	Orders []BatchCancelOrderEntry `json:"orders"`
}

// AmendOrderRequest is the request body for POST /portfolio/orders/{id}/amend.
type AmendOrderRequest struct {
	Ticker               string `json:"ticker"`
	Side                 Side   `json:"side"`
	Action               Action `json:"action"`
	Subaccount           int    `json:"subaccount,omitempty"`
	ClientOrderID        string `json:"client_order_id,omitempty"`
	UpdatedClientOrderID string `json:"updated_client_order_id,omitempty"`
	YesPriceDollars      string `json:"yes_price_dollars,omitempty"`
	NoPriceDollars       string `json:"no_price_dollars,omitempty"`
	CountFP              string `json:"count_fp,omitempty"`
}

// DecreaseOrderRequest is the request body for POST /portfolio/orders/{id}/decrease.
type DecreaseOrderRequest struct {
	Subaccount int    `json:"subaccount,omitempty"`
	ReduceByFP string `json:"reduce_by_fp,omitempty"`
	ReduceToFP string `json:"reduce_to_fp,omitempty"`
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// OrderResponse is the full order object returned by the Kalshi API.
type OrderResponse struct {
	OrderID                 string      `json:"order_id"`
	UserID                  string      `json:"user_id"`
	ClientOrderID           string      `json:"client_order_id"`
	Ticker                  string      `json:"ticker"`
	Side                    Side        `json:"side"`
	Action                  Action      `json:"action"`
	Type                    OrderType   `json:"type"`
	Status                  OrderStatus `json:"status"`
	YesPriceDollars         string      `json:"yes_price_dollars"`
	NoPriceDollars          string      `json:"no_price_dollars"`
	FillCountFP             string      `json:"fill_count_fp"`
	RemainingCountFP        string      `json:"remaining_count_fp"`
	InitialCountFP          string      `json:"initial_count_fp"`
	TakerFillCostDollars    string      `json:"taker_fill_cost_dollars"`
	MakerFillCostDollars    string      `json:"maker_fill_cost_dollars"`
	TakerFeesDollars        string      `json:"taker_fees_dollars"`
	MakerFeesDollars        string      `json:"maker_fees_dollars"`
	ExpirationTime          string      `json:"expiration_time"`
	CreatedTime             string      `json:"created_time"`
	LastUpdateTime          string      `json:"last_update_time"`
	SelfTradePreventionType STPType     `json:"self_trade_prevention_type"`
	OrderGroupID            string      `json:"order_group_id"`
	CancelOrderOnPause      bool        `json:"cancel_order_on_pause"`
	SubaccountNumber        int         `json:"subaccount_number"`
}

// BatchCreateEntry is a single entry in the batch create orders response.
type BatchCreateEntry struct {
	ClientOrderID string         `json:"client_order_id"`
	Order         *OrderResponse `json:"order"`
	Error         *APIErrorBody  `json:"error"`
}

// BatchCreateResponse is the response from POST /portfolio/orders/batched.
type BatchCreateResponse struct {
	Orders []BatchCreateEntry `json:"orders"`
}

// BatchCancelEntry is a single entry in the batch cancel orders response.
type BatchCancelEntry struct {
	OrderID     string         `json:"order_id"`
	ReducedByFP string         `json:"reduced_by_fp"`
	Order       *OrderResponse `json:"order"`
	Error       *APIErrorBody  `json:"error"`
}

// BatchCancelResponse is the response from DELETE /portfolio/orders/batched.
type BatchCancelResponse struct {
	Orders []BatchCancelEntry `json:"orders"`
}

// APIErrorBody is the error object returned inside batch responses.
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
	Service string `json:"service"`
}

// SingleCreateResponse is the response from POST /portfolio/orders (single order)
// and GET /portfolio/orders/{id}.
type SingleCreateResponse struct {
	Order OrderResponse `json:"order"`
}

// CancelOrderResponse is the response from DELETE /portfolio/orders/{id}.
type CancelOrderResponse struct {
	Order       OrderResponse `json:"order"`
	ReducedByFP string        `json:"reduced_by_fp"`
}

// AmendOrderResponse is the response from POST /portfolio/orders/{id}/amend.
type AmendOrderResponse struct {
	OldOrder OrderResponse `json:"old_order"`
	Order    OrderResponse `json:"order"`
}

// QueuePositionEntry is a single entry in the queue positions response.
type QueuePositionEntry struct {
	OrderID         string `json:"order_id"`
	MarketTicker    string `json:"market_ticker"`
	QueuePositionFP string `json:"queue_position_fp"`
}

// GetQueuePositionsResponse is the response from GET /portfolio/orders/queue_positions.
type GetQueuePositionsResponse struct {
	QueuePositions []QueuePositionEntry `json:"queue_positions"`
}

// GetQueuePositionResponse is the response from GET /portfolio/orders/{id}/queue_position.
type GetQueuePositionResponse struct {
	QueuePositionFP string `json:"queue_position_fp"`
}

// GetOrdersResponse is the paginated response from GET /portfolio/orders.
type GetOrdersResponse struct {
	Orders []OrderResponse `json:"orders"`
	Cursor string          `json:"cursor"`
}
