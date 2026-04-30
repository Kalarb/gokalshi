package gokalshi

import "context"

// HTTPClient defines the contract for the Kalshi HTTP API client.
// Consumers should type-annotate against this interface, not *Client directly.
type HTTPClient interface {
	// Account
	GetAccountAPILimits(ctx context.Context) (GetAccountApiLimitsResponse, error)

	// Exchange
	GetExchangeStatus(ctx context.Context) (ExchangeStatus, error)
	GetExchangeAnnouncements(ctx context.Context) (GetExchangeAnnouncementsResponse, error)
	GetExchangeSchedule(ctx context.Context) (GetExchangeScheduleResponse, error)
	GetUserDataTimestamp(ctx context.Context) (GetUserDataTimestampResponse, error)
	GetSeriesFeeChanges(ctx context.Context, params GetSeriesFeeChangesParams) (GetSeriesFeeChangesResponse, error)

	// Orders
	CreateOrder(ctx context.Context, req CreateOrderRequest) (CreateOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (CancelOrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (CreateOrderResponse, error)
	GetOrders(ctx context.Context, params GetOrdersParams) (GetOrdersResponse, error)
	BatchCreateOrders(ctx context.Context, orders []CreateOrderRequest) (BatchCreateOrdersResponse, error)
	BatchCancelOrders(ctx context.Context, orders []BatchCancelOrdersRequestOrder) (BatchCancelOrdersResponse, error)
	AmendOrder(ctx context.Context, orderID string, req AmendOrderRequest) (AmendOrderResponse, error)
	DecreaseOrder(ctx context.Context, orderID string, req DecreaseOrderRequest) (CreateOrderResponse, error)
	GetQueuePositions(ctx context.Context, params GetQueuePositionsParams) (GetOrderQueuePositionsResponse, error)
	GetQueuePosition(ctx context.Context, orderID string) (GetOrderQueuePositionResponse, error)

	// Portfolio
	GetBalance(ctx context.Context) (GetBalanceResponse, error)
	GetPositions(ctx context.Context, params GetPositionsParams) (GetPositionsResponse, error)
	GetFills(ctx context.Context, params GetFillsParams) (GetFillsResponse, error)
	GetSettlements(ctx context.Context, params GetSettlementsParams) (GetSettlementsResponse, error)

	// Markets
	GetMarketOrderbook(ctx context.Context, ticker string, params GetOrderbookParams) (GetMarketOrderbookResponse, error)
	GetMarketOrderbooks(ctx context.Context, params GetMarketOrderbooksParams) (GetMarketOrderbooksResponse, error)
	GetTrades(ctx context.Context, params GetTradesParams) (GetTradesResponse, error)
	GetMarket(ctx context.Context, ticker string) (GetMarketResponse, error)
	GetMarkets(ctx context.Context, params GetMarketsParams) (GetMarketsResponse, error)
	GetMarketCandlesticks(ctx context.Context, seriesTicker, ticker string, params GetMarketCandlesticksParams) (GetMarketCandlesticksResponse, error)
	GetBatchMarketCandlesticks(ctx context.Context, params GetBatchMarketCandlesticksParams) (BatchGetMarketCandlesticksResponse, error)

	// Events
	GetEvent(ctx context.Context, eventTicker string, params GetEventParams) (GetEventResponse, error)
	GetEvents(ctx context.Context, params GetEventsParams) (GetEventsResponse, error)
	GetEventMetadata(ctx context.Context, eventTicker string) (GetEventMetadataResponse, error)
	GetMultivariateEvents(ctx context.Context, params GetMultivariateEventsParams) (GetMultivariateEventsResponse, error)
	GetEventCandlesticks(ctx context.Context, seriesTicker, eventTicker string, params GetEventCandlesticksParams) (GetEventCandlesticksResponse, error)
	GetEventForecastPercentileHistory(ctx context.Context, seriesTicker, eventTicker string, params GetEventForecastPercentileHistoryParams) (GetEventForecastPercentilesHistoryResponse, error)

	// Series
	GetSeries(ctx context.Context, seriesTicker string, params GetSeriesParams) (GetSeriesResponse, error)
	GetSeriesList(ctx context.Context, params GetSeriesListParams) (GetSeriesListResponse, error)

	// Search
	GetTagsByCategories(ctx context.Context) (GetTagsForSeriesCategoriesResponse, error)
	GetFiltersBySport(ctx context.Context) (GetFiltersBySportsResponse, error)

	// Lifecycle
	Close()
}

// WebSocketClient defines the contract for the Kalshi WebSocket client.
type WebSocketClient interface {
	Connect(ctx context.Context) error
	Close() error
	MsgCh() <-chan []byte
	ListenLoop(ctx context.Context)
	AddMarkets(ctx context.Context, tickers []string, channels []string) error
	RemoveMarkets(ctx context.Context, tickers []string, channels []string) error
}

// Compile-time interface satisfaction checks.
var (
	_ HTTPClient      = (*Client)(nil)
	_ WebSocketClient = (*WSClient)(nil)
)
