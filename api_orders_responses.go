package gokalshi

// Response types for order-related API endpoints.

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
