package gokalshi

// Kalshi API request structures.

// ---------------------------------------------------------------------------
// Order creation
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

// ---------------------------------------------------------------------------
// Order cancellation
// ---------------------------------------------------------------------------

// BatchCancelOrderEntry is a single entry in a batch cancel request.
type BatchCancelOrderEntry struct {
	OrderID    string `json:"order_id"`
	Subaccount int    `json:"subaccount,omitempty"`
}

// BatchCancelRequest is the request body for DELETE /portfolio/orders/batched.
type BatchCancelRequest struct {
	Orders []BatchCancelOrderEntry `json:"orders"`
}

// ---------------------------------------------------------------------------
// Order amendment
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Order decrease
// ---------------------------------------------------------------------------

// DecreaseOrderRequest is the request body for POST /portfolio/orders/{id}/decrease.
type DecreaseOrderRequest struct {
	Subaccount int    `json:"subaccount,omitempty"`
	ReduceByFP string `json:"reduce_by_fp,omitempty"`
	ReduceToFP string `json:"reduce_to_fp,omitempty"`
}
