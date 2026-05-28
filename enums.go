package gokalshi

// Typed string enums for Kalshi API request/response fields.
// Using string-based enums so encoding/json serializes them without custom marshalers.

// ---------------------------------------------------------------------------
// Core trading enums
// ---------------------------------------------------------------------------

// Side represents the contract side: yes or no.
type Side string

const (
	SideYes Side = "yes"
	SideNo  Side = "no"
)

// Action represents a trade direction: buy or sell.
type Action string

const (
	ActionBuy  Action = "buy"
	ActionSell Action = "sell"
)

// OrderStatus represents the lifecycle status of an order.
type OrderStatus string

const (
	OrderStatusResting  OrderStatus = "resting"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusExecuted OrderStatus = "executed"
)

// OrderType represents the type of an order.
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// TimeInForce represents the time-in-force policy for an order.
type TimeInForce string

const (
	TimeInForceFOK TimeInForce = "fill_or_kill"
	TimeInForceGTC TimeInForce = "good_till_canceled"
	TimeInForceIOC TimeInForce = "immediate_or_cancel"
)

// STPType represents the self-trade prevention strategy.
type STPType string

const (
	STPTakerAtCross STPType = "taker_at_cross"
	STPMaker        STPType = "maker"
)

// SelfTradePreventionType is an alias for STPType (spec-canonical name).
type SelfTradePreventionType = STPType

// BookSide represents the side of the order book.
type BookSide string

const (
	BookSideBid BookSide = "bid"
	BookSideAsk BookSide = "ask"
)

// ---------------------------------------------------------------------------
// Market enums
// ---------------------------------------------------------------------------

// MarketStatus represents the lifecycle status of a market.
type MarketStatus string

const (
	MarketStatusInitialized MarketStatus = "initialized"
	MarketStatusInactive    MarketStatus = "inactive"
	MarketStatusActive      MarketStatus = "active"
	MarketStatusClosed      MarketStatus = "closed"
	MarketStatusDetermined  MarketStatus = "determined"
	MarketStatusDisputed    MarketStatus = "disputed"
	MarketStatusAmended     MarketStatus = "amended"
	MarketStatusFinalized   MarketStatus = "finalized"
)

// MarketResult represents the settlement result of a market.
type MarketResult string

const (
	MarketResultYes    MarketResult = "yes"
	MarketResultNo     MarketResult = "no"
	MarketResultScalar MarketResult = "scalar"
	MarketResultVoid   MarketResult = "void"
	MarketResultNone   MarketResult = ""
)

// MarketType represents whether a market is binary or scalar.
type MarketType string

const (
	MarketTypeBinary MarketType = "binary"
	MarketTypeScalar MarketType = "scalar"
)

// FeeType represents the fee structure for a series.
type FeeType string

const (
	FeeTypeQuadratic          FeeType = "quadratic"
	FeeTypeQuadraticWithMaker FeeType = "quadratic_with_maker_fees"
	FeeTypeFlat               FeeType = "flat"
)

// CollateralReturnType represents how collateral is returned for an event.
type CollateralReturnType string

const (
	CollateralReturnMECNET   CollateralReturnType = "MECNET"
	CollateralReturnDIRECNET CollateralReturnType = "DIRECNET"
	CollateralReturnNone     CollateralReturnType = ""
)

// ---------------------------------------------------------------------------
// Exchange / announcement enums
// ---------------------------------------------------------------------------

// AnnouncementType represents the severity of an exchange announcement.
type AnnouncementType string

const (
	AnnouncementTypeInfo    AnnouncementType = "info"
	AnnouncementTypeWarning AnnouncementType = "warning"
	AnnouncementTypeError   AnnouncementType = "error"
)

// AnnouncementStatus represents whether an announcement is active.
type AnnouncementStatus string

const (
	AnnouncementStatusActive   AnnouncementStatus = "active"
	AnnouncementStatusInactive AnnouncementStatus = "inactive"
)

// ---------------------------------------------------------------------------
// WebSocket enums
// ---------------------------------------------------------------------------

// MarketLifecycleEventType represents an event in a market's lifecycle.
type MarketLifecycleEventType string

const (
	MLCreated                  MarketLifecycleEventType = "created"
	MLActivated                MarketLifecycleEventType = "activated"
	MLDeactivated              MarketLifecycleEventType = "deactivated"
	MLCloseDateUpdated         MarketLifecycleEventType = "close_date_updated"
	MLDetermined               MarketLifecycleEventType = "determined"
	MLSettled                  MarketLifecycleEventType = "settled"
	MLFractionalTradingUpdated MarketLifecycleEventType = "fractional_trading_updated"
	MLPriceLevelStructUpdated  MarketLifecycleEventType = "price_level_structure_updated"
)

// OrderGroupEventType represents an event in an order group's lifecycle.
type OrderGroupEventType string

const (
	OGCreated      OrderGroupEventType = "created"
	OGTriggered    OrderGroupEventType = "triggered"
	OGReset        OrderGroupEventType = "reset"
	OGDeleted      OrderGroupEventType = "deleted"
	OGLimitUpdated OrderGroupEventType = "limit_updated"
)

// WSCmd represents a WebSocket command type.
type WSCmd string

const (
	WSCmdSubscribe          WSCmd = "subscribe"
	WSCmdUnsubscribe        WSCmd = "unsubscribe"
	WSCmdUpdateSubscription WSCmd = "update_subscription"
	WSCmdListSubscriptions  WSCmd = "list_subscriptions"
)

// WSUpdateAction represents the action for an update_subscription command.
type WSUpdateAction string

const (
	WSUpdateAddMarkets    WSUpdateAction = "add_markets"
	WSUpdateDeleteMarkets WSUpdateAction = "delete_markets"
)

// WSMessageType represents the type field of an incoming WebSocket message.
type WSMessageType string

const (
	WSMsgOrderbookSnapshot           WSMessageType = "orderbook_snapshot"
	WSMsgOrderbookDelta              WSMessageType = "orderbook_delta"
	WSMsgTicker                      WSMessageType = "ticker"
	WSMsgTrade                       WSMessageType = "trade"
	WSMsgFill                        WSMessageType = "fill"
	WSMsgMarketPosition              WSMessageType = "market_position"
	WSMsgMarketLifecycleV2           WSMessageType = "market_lifecycle_v2"
	WSMsgEventLifecycle              WSMessageType = "event_lifecycle"
	WSMsgMultivariateMarketLifecycle WSMessageType = "multivariate_market_lifecycle"
	WSMsgMultivariateLookup          WSMessageType = "multivariate_lookup"
	WSMsgUserOrder                   WSMessageType = "user_order"
	WSMsgOrderGroupUpdates           WSMessageType = "order_group_updates"
	WSMsgRFQCreated                  WSMessageType = "rfq_created"
	WSMsgRFQDeleted                  WSMessageType = "rfq_deleted"
	WSMsgQuoteCreated                WSMessageType = "quote_created"
	WSMsgQuoteAccepted               WSMessageType = "quote_accepted"
	WSMsgQuoteExecuted               WSMessageType = "quote_executed"
	WSMsgEventFeeUpdate              WSMessageType = "event_fee_update"
)

// WSResponseType represents the type field of a WebSocket command response.
type WSResponseType string

const (
	WSRespSubscribed   WSResponseType = "subscribed"
	WSRespOk           WSResponseType = "ok"
	WSRespUnsubscribed WSResponseType = "unsubscribed"
	WSRespError        WSResponseType = "error"
)
