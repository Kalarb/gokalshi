package gokalshi

// Kalshi WebSocket API message types.
// Centralized here as the single source of truth for all WS command and response parsing.

// ---------------------------------------------------------------------------
// Outgoing commands
// ---------------------------------------------------------------------------

// WSCommand is the generic envelope for outgoing WebSocket commands.
// Cmd values: see WSCmd constants.
type WSCommand struct {
	ID     int   `json:"id"`
	Cmd    WSCmd `json:"cmd"`
	Params any   `json:"params,omitempty"` // nil for list_subscriptions
}

// SubscribeParams are the parameters for a "subscribe" command.
type SubscribeParams struct {
	Channels            []string `json:"channels,omitempty"`
	MarketTicker        string   `json:"market_ticker,omitempty"`
	MarketTickers       []string `json:"market_tickers,omitempty"`
	MarketID            string   `json:"market_id,omitempty"`
	MarketIDs           []string `json:"market_ids,omitempty"`
	SendInitialSnapshot bool     `json:"send_initial_snapshot,omitempty"`
	SkipTickerAck       bool     `json:"skip_ticker_ack,omitempty"`
	ShardFactor         int      `json:"shard_factor,omitempty"`
	ShardKey            int      `json:"shard_key,omitempty"`
}

// UnsubscribeParams are the parameters for an "unsubscribe" command.
type UnsubscribeParams struct {
	SIDs []int `json:"sids"`
}

// UpdateSubParams are the parameters for an "update_subscription" command.
type UpdateSubParams struct {
	SID                 int            `json:"sid,omitempty"`
	SIDs                []int          `json:"sids,omitempty"`
	MarketTicker        string         `json:"market_ticker,omitempty"`
	MarketTickers       []string       `json:"market_tickers,omitempty"`
	MarketID            string         `json:"market_id,omitempty"`
	MarketIDs           []string       `json:"market_ids,omitempty"`
	Action              WSUpdateAction `json:"action"`
	SendInitialSnapshot bool           `json:"send_initial_snapshot,omitempty"`
}

// ---------------------------------------------------------------------------
// Incoming response msg bodies (parsed from WSMessage.Msg)
// ---------------------------------------------------------------------------

// SubscribedBody is the msg body for a "subscribed" response.
// The SID is nested here, NOT at the top level of the WS message.
type SubscribedBody struct {
	Channel string `json:"channel"`
	SID     int    `json:"sid"`
}

// OkBody is the msg body for an "ok" response.
type OkBody struct {
	MarketTickers []string `json:"market_tickers"`
	MarketIDs     []string `json:"market_ids"`
}

// WSErrorBody is the msg body for an "error" response.
type WSErrorBody struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	MarketID     string `json:"market_id,omitempty"`
	MarketTicker string `json:"market_ticker,omitempty"`
}

// ---------------------------------------------------------------------------
// Incoming channel data messages
// ---------------------------------------------------------------------------

// OrderbookSnapshotData is the msg body for an "orderbook_snapshot" message.
type OrderbookSnapshotData struct {
	MarketTicker string     `json:"market_ticker"`
	MarketID     string     `json:"market_id"`
	YesDollarsFP [][]string `json:"yes_dollars_fp"`
	NoDollarsFP  [][]string `json:"no_dollars_fp"`
}

// OrderbookDeltaData is the msg body for an "orderbook_delta" message.
type OrderbookDeltaData struct {
	MarketTicker  string `json:"market_ticker"`
	MarketID      string `json:"market_id"`
	PriceDollars  string `json:"price_dollars"`
	DeltaFP       string `json:"delta_fp"`
	Side          Side   `json:"side"`
	ClientOrderID string `json:"client_order_id,omitempty"`
	Subaccount    int    `json:"subaccount,omitempty"`
	Ts            string `json:"ts,omitempty"`
	TsMs          int64  `json:"ts_ms,omitempty"`
}

// MarketLifecycleAdditionalMetadata is optional metadata sent when a market is created.
type MarketLifecycleAdditionalMetadata struct {
	Name                 string   `json:"name,omitempty"`
	Title                string   `json:"title,omitempty"`
	YesSubTitle          string   `json:"yes_sub_title,omitempty"`
	NoSubTitle           string   `json:"no_sub_title,omitempty"`
	RulesPrimary         string   `json:"rules_primary,omitempty"`
	RulesSecondary       string   `json:"rules_secondary,omitempty"`
	CanCloseEarly        bool     `json:"can_close_early,omitempty"`
	EventTicker          string   `json:"event_ticker,omitempty"`
	ExpectedExpirationTs int64    `json:"expected_expiration_ts,omitempty"`
	StrikeType           string   `json:"strike_type,omitempty"`
	FloorStrike          *float64 `json:"floor_strike,omitempty"`
	CapStrike            *float64 `json:"cap_strike,omitempty"`
	CustomStrike         any      `json:"custom_strike,omitempty"`
}

// MarketLifecycleV2Data is the msg body for a "market_lifecycle_v2" message.
type MarketLifecycleV2Data struct {
	EventType                MarketLifecycleEventType           `json:"event_type"`
	MarketTicker             string                             `json:"market_ticker"`
	OpenTs                   int64                              `json:"open_ts,omitempty"`
	CloseTs                  int64                              `json:"close_ts,omitempty"`
	Result                   string                             `json:"result,omitempty"`
	DeterminationTs          int64                              `json:"determination_ts,omitempty"`
	SettlementValue          string                             `json:"settlement_value,omitempty"`
	SettledTs                int64                              `json:"settled_ts,omitempty"`
	IsDeactivated            *bool                              `json:"is_deactivated,omitempty"`
	FractionalTradingEnabled *bool                              `json:"fractional_trading_enabled,omitempty"`
	PriceLevelStructure      string                             `json:"price_level_structure,omitempty"`
	AdditionalMetadata       *MarketLifecycleAdditionalMetadata `json:"additional_metadata,omitempty"`
}

// EventLifecycleData is the msg body for an "event_lifecycle" message.
type EventLifecycleData struct {
	EventTicker          string               `json:"event_ticker"`
	Title                string               `json:"title"`
	Subtitle             string               `json:"subtitle"`
	CollateralReturnType CollateralReturnType `json:"collateral_return_type"`
	SeriesTicker         string               `json:"series_ticker"`
	StrikeDate           int64                `json:"strike_date,omitempty"`
	StrikePeriod         string               `json:"strike_period,omitempty"`
}

// MultivariateLookupSelectedMarket is a single leg in a multivariate lookup.
type MultivariateLookupSelectedMarket struct {
	EventTicker  string `json:"event_ticker"`
	MarketTicker string `json:"market_ticker"`
	Side         Side   `json:"side"`
}

// MultivariateLookupData is the msg body for a "multivariate_lookup" message.
type MultivariateLookupData struct {
	CollectionTicker string                             `json:"collection_ticker"`
	EventTicker      string                             `json:"event_ticker"`
	MarketTicker     string                             `json:"market_ticker"`
	SelectedMarkets  []MultivariateLookupSelectedMarket `json:"selected_markets"`
}

// MarketPositionData is the msg body for a "market_position" message.
type MarketPositionData struct {
	UserID                 string `json:"user_id"`
	MarketTicker           string `json:"market_ticker"`
	PositionFP             string `json:"position_fp"`
	PositionCostDollars    string `json:"position_cost_dollars"`
	RealizedPnlDollars     string `json:"realized_pnl_dollars"`
	FeesPaidDollars        string `json:"fees_paid_dollars"`
	PositionFeeCostDollars string `json:"position_fee_cost_dollars"`
	VolumeFP               string `json:"volume_fp"`
	Subaccount             int    `json:"subaccount,omitempty"`
}

// FillData is the msg body for a "fill" message.
type FillData struct {
	TradeID         string `json:"trade_id"`
	OrderID         string `json:"order_id"`
	MarketTicker    string `json:"market_ticker"`
	IsTaker         bool   `json:"is_taker"`
	Side            Side   `json:"side"`
	Action          Action `json:"action"`
	YesPriceDollars string `json:"yes_price_dollars"`
	CountFP         string `json:"count_fp"`
	FeeCost         string `json:"fee_cost"`
	Ts              int64  `json:"ts"`
	TsMs            int64  `json:"ts_ms,omitempty"`
	ClientOrderID   string `json:"client_order_id,omitempty"`
	PostPositionFP  string `json:"post_position_fp,omitempty"`
	PurchasedSide   Side   `json:"purchased_side,omitempty"`
	Subaccount      int    `json:"subaccount,omitempty"`
}

// TradeData is the msg body for a "trade" message.
type TradeData struct {
	TradeID         string `json:"trade_id"`
	MarketTicker    string `json:"market_ticker"`
	YesPriceDollars string `json:"yes_price_dollars"`
	NoPriceDollars  string `json:"no_price_dollars"`
	CountFP         string `json:"count_fp"`
	TakerSide       Side   `json:"taker_side"`
	Ts              int64  `json:"ts"`
	TsMs            int64  `json:"ts_ms,omitempty"`
}

// TickerData is the msg body for a "ticker" message.
type TickerData struct {
	MarketTicker       string `json:"market_ticker"`
	MarketID           string `json:"market_id"`
	PriceDollars       string `json:"price_dollars,omitempty"`
	YesBidDollars      string `json:"yes_bid_dollars,omitempty"`
	YesAskDollars      string `json:"yes_ask_dollars,omitempty"`
	YesBidSizeFP       string `json:"yes_bid_size_fp,omitempty"`
	YesAskSizeFP       string `json:"yes_ask_size_fp,omitempty"`
	LastTradeSizeFP    string `json:"last_trade_size_fp,omitempty"`
	VolumeFP           string `json:"volume_fp,omitempty"`
	OpenInterestFP     string `json:"open_interest_fp,omitempty"`
	DollarVolume       int    `json:"dollar_volume,omitempty"`
	DollarOpenInterest int    `json:"dollar_open_interest,omitempty"`
	Ts                 int64  `json:"ts,omitempty"`
	TsMs               int64  `json:"ts_ms,omitempty"`
	Time               string `json:"time,omitempty"`
}

// UserOrderData is the msg body for a "user_order" message.
type UserOrderData struct {
	OrderID                 string      `json:"order_id"`
	UserID                  string      `json:"user_id"`
	Ticker                  string      `json:"ticker"`
	Status                  OrderStatus `json:"status"`
	Side                    Side        `json:"side"`
	IsYes                   bool        `json:"is_yes"`
	YesPriceDollars         string      `json:"yes_price_dollars"`
	FillCountFP             string      `json:"fill_count_fp"`
	RemainingCountFP        string      `json:"remaining_count_fp"`
	InitialCountFP          string      `json:"initial_count_fp"`
	TakerFillCostDollars    string      `json:"taker_fill_cost_dollars"`
	MakerFillCostDollars    string      `json:"maker_fill_cost_dollars"`
	TakerFeesDollars        string      `json:"taker_fees_dollars"`
	MakerFeesDollars        string      `json:"maker_fees_dollars"`
	ClientOrderID           string      `json:"client_order_id,omitempty"`
	OrderGroupID            string      `json:"order_group_id,omitempty"`
	SelfTradePreventionType STPType     `json:"self_trade_prevention_type,omitempty"`
	CreatedTime             string      `json:"created_time"`
	CreatedTsMs             int64       `json:"created_ts_ms,omitempty"`
	LastUpdateTime          string      `json:"last_update_time,omitempty"`
	LastUpdatedTsMs         int64       `json:"last_updated_ts_ms,omitempty"`
	ExpirationTime          string      `json:"expiration_time,omitempty"`
	ExpirationTsMs          int64       `json:"expiration_ts_ms,omitempty"`
	SubaccountNumber        int         `json:"subaccount_number,omitempty"`
}

// OrderGroupUpdateData is the msg body for an "order_group_updates" message.
type OrderGroupUpdateData struct {
	EventType        OrderGroupEventType `json:"event_type"`
	OrderGroupID     string              `json:"order_group_id"`
	ContractsLimitFP string              `json:"contracts_limit_fp,omitempty"`
}

// ---------------------------------------------------------------------------
// Communications channel (RFQ / quotes)
// ---------------------------------------------------------------------------

// RFQCreatedData is the msg body for an "rfq_created" message.
type RFQCreatedData struct {
	ID                  string           `json:"id"`
	CreatorID           string           `json:"creator_id"`
	MarketTicker        string           `json:"market_ticker"`
	EventTicker         string           `json:"event_ticker,omitempty"`
	ContractsFP         string           `json:"contracts_fp,omitempty"`
	TargetCostDollars   string           `json:"target_cost_dollars,omitempty"`
	CreatedTs           string           `json:"created_ts"`
	MVECollectionTicker string           `json:"mve_collection_ticker,omitempty"`
	MVESelectedLegs     []MVESelectedLeg `json:"mve_selected_legs,omitempty"`
}

// RFQDeletedData is the msg body for an "rfq_deleted" message.
type RFQDeletedData struct {
	ID                string `json:"id"`
	CreatorID         string `json:"creator_id"`
	MarketTicker      string `json:"market_ticker"`
	EventTicker       string `json:"event_ticker,omitempty"`
	ContractsFP       string `json:"contracts_fp,omitempty"`
	TargetCostDollars string `json:"target_cost_dollars,omitempty"`
	DeletedTs         string `json:"deleted_ts"`
}

// QuoteCreatedData is the msg body for a "quote_created" message.
type QuoteCreatedData struct {
	QuoteID               string `json:"quote_id"`
	RFQID                 string `json:"rfq_id"`
	QuoteCreatorID        string `json:"quote_creator_id"`
	MarketTicker          string `json:"market_ticker"`
	EventTicker           string `json:"event_ticker,omitempty"`
	YesBidDollars         string `json:"yes_bid_dollars"`
	NoBidDollars          string `json:"no_bid_dollars"`
	YesContractsOfferedFP string `json:"yes_contracts_offered_fp,omitempty"`
	NoContractsOfferedFP  string `json:"no_contracts_offered_fp,omitempty"`
	RFQTargetCostDollars  string `json:"rfq_target_cost_dollars,omitempty"`
	CreatedTs             string `json:"created_ts"`
}

// QuoteAcceptedData is the msg body for a "quote_accepted" message.
type QuoteAcceptedData struct {
	QuoteID               string `json:"quote_id"`
	RFQID                 string `json:"rfq_id"`
	QuoteCreatorID        string `json:"quote_creator_id"`
	MarketTicker          string `json:"market_ticker"`
	EventTicker           string `json:"event_ticker,omitempty"`
	YesBidDollars         string `json:"yes_bid_dollars"`
	NoBidDollars          string `json:"no_bid_dollars"`
	AcceptedSide          Side   `json:"accepted_side,omitempty"`
	ContractsAcceptedFP   string `json:"contracts_accepted_fp,omitempty"`
	YesContractsOfferedFP string `json:"yes_contracts_offered_fp,omitempty"`
	NoContractsOfferedFP  string `json:"no_contracts_offered_fp,omitempty"`
	RFQTargetCostDollars  string `json:"rfq_target_cost_dollars,omitempty"`
}

// QuoteExecutedData is the msg body for a "quote_executed" message.
type QuoteExecutedData struct {
	QuoteID        string `json:"quote_id"`
	RFQID          string `json:"rfq_id"`
	QuoteCreatorID string `json:"quote_creator_id"`
	RFQCreatorID   string `json:"rfq_creator_id"`
	OrderID        string `json:"order_id"`
	ClientOrderID  string `json:"client_order_id"`
	MarketTicker   string `json:"market_ticker"`
	ExecutedTs     string `json:"executed_ts"`
}

// ---------------------------------------------------------------------------
// WS error codes (from Kalshi API spec)
// ---------------------------------------------------------------------------

const (
	WSErrUnableToProcess     = 1
	WSErrParamsRequired      = 2
	WSErrChannelsRequired    = 3
	WSErrSIDsRequired        = 4
	WSErrUnknownCommand      = 5
	WSErrAlreadySubscribed   = 6
	WSErrUnknownSID          = 7
	WSErrUnknownChannel      = 8
	WSErrAuthRequired        = 9
	WSErrChannelError        = 10
	WSErrInvalidParam        = 11
	WSErrExactlyOneSID       = 12
	WSErrUnsupportedAction   = 13
	WSErrMarketTickerReq     = 14
	WSErrActionRequired      = 15
	WSErrMarketNotFound      = 16
	WSErrInternal            = 17
	WSErrCommandTimeout      = 18
	WSErrShardFactorPositive = 19
	WSErrShardFactorRequired = 20
	WSErrShardKeyRange       = 21
	WSErrShardFactorTooLarge = 22
)
