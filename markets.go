package gokalshi

import (
	"context"
	"fmt"
)

const pathMarkets = "/trade-api/v2/markets"

// GetMarketOrderbook retrieves the orderbook for a market.
func (c *Client) GetMarketOrderbook(ctx context.Context, ticker string, params GetOrderbookParams) (GetMarketOrderbookResponse, error) {
	path := fmt.Sprintf("%s/%s/orderbook", pathMarkets, ticker)
	return getJSON[GetMarketOrderbookResponse](c, ctx, path, params.toMap())
}

// GetMarketOrderbooks retrieves orderbooks for multiple markets in a single request.
func (c *Client) GetMarketOrderbooks(ctx context.Context, params GetMarketOrderbooksParams) (GetMarketOrderbooksResponse, error) {
	return getJSON[GetMarketOrderbooksResponse](c, ctx, pathMarkets+"/orderbooks", params.toMap())
}

// GetTrades retrieves recent trades.
func (c *Client) GetTrades(ctx context.Context, params GetTradesParams) (GetTradesResponse, error) {
	return getJSON[GetTradesResponse](c, ctx, pathMarkets+"/trades", params.toMap())
}

// GetMarket retrieves details for a single market.
func (c *Client) GetMarket(ctx context.Context, ticker string) (MarketResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMarkets, ticker)
	return getJSON[MarketResponse](c, ctx, path, nil)
}

// GetMarkets retrieves markets matching the given parameters.
func (c *Client) GetMarkets(ctx context.Context, params GetMarketsParams) (GetMarketsResponse, error) {
	return getJSON[GetMarketsResponse](c, ctx, pathMarkets, params.toMap())
}

// GetMarketCandlesticks retrieves candlestick data for a single market.
func (c *Client) GetMarketCandlesticks(ctx context.Context, seriesTicker, ticker string, params GetMarketCandlesticksParams) (GetMarketCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/markets/%s/candlesticks", seriesTicker, ticker)
	return getJSON[GetMarketCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetBatchMarketCandlesticks retrieves candlestick data for multiple markets.
func (c *Client) GetBatchMarketCandlesticks(ctx context.Context, params GetBatchMarketCandlesticksParams) (GetBatchMarketCandlesticksResponse, error) {
	return getJSON[GetBatchMarketCandlesticksResponse](c, ctx, pathMarkets+"/candlesticks", params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetOrderbookParams holds optional query parameters for GetMarketOrderbook.
type GetOrderbookParams struct {
	Depth int
}

func (p GetOrderbookParams) toMap() map[string]string {
	return NewQuery().
		Int("depth", p.Depth).
		Build()
}

// GetMarketOrderbooksParams holds query parameters for GetMarketOrderbooks.
type GetMarketOrderbooksParams struct {
	Tickers string // comma-separated, 1-100 tickers
}

func (p GetMarketOrderbooksParams) toMap() map[string]string {
	return NewQuery().
		String("tickers", p.Tickers).
		Build()
}

// GetTradesParams holds optional query parameters for GetTrades.
type GetTradesParams struct {
	Ticker string
	Limit  int
	Cursor string
	MinTs  int64
	MaxTs  int64
}

func (p GetTradesParams) toMap() map[string]string {
	return NewQuery().
		String("ticker", p.Ticker).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Int64("min_ts", p.MinTs).
		Int64("max_ts", p.MaxTs).
		Build()
}

// GetMarketsParams holds optional query parameters for GetMarkets.
type GetMarketsParams struct {
	Limit        int
	Cursor       string
	EventTicker  string
	SeriesTicker string
	Status       MarketStatus
	Tickers      string
	MVEFilter    string
	MinCreatedTs int64
	MaxCreatedTs int64
	MinUpdatedTs int64
	MinCloseTs   int64
	MaxCloseTs   int64
	MinSettledTs int64
	MaxSettledTs int64
}

func (p GetMarketsParams) toMap() map[string]string {
	return NewQuery().
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		String("event_ticker", p.EventTicker).
		String("series_ticker", p.SeriesTicker).
		String("status", string(p.Status)).
		String("tickers", p.Tickers).
		String("mve_filter", p.MVEFilter).
		Int64("min_created_ts", p.MinCreatedTs).
		Int64("max_created_ts", p.MaxCreatedTs).
		Int64("min_updated_ts", p.MinUpdatedTs).
		Int64("min_close_ts", p.MinCloseTs).
		Int64("max_close_ts", p.MaxCloseTs).
		Int64("min_settled_ts", p.MinSettledTs).
		Int64("max_settled_ts", p.MaxSettledTs).
		Build()
}

// GetMarketCandlesticksParams holds query parameters for GetCandlesticks.
type GetMarketCandlesticksParams struct {
	StartTs                  int64
	EndTs                    int64
	PeriodInterval           int // 1, 60, or 1440
	IncludeLatestBeforeStart bool
}

func (p GetMarketCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Bool("include_latest_before_start", p.IncludeLatestBeforeStart).
		Build()
}

// GetBatchMarketCandlesticksParams holds query parameters for GetBatchCandlesticks.
type GetBatchMarketCandlesticksParams struct {
	MarketTickers            string // comma-separated, max 100
	StartTs                  int64
	EndTs                    int64
	PeriodInterval           int
	IncludeLatestBeforeStart bool
}

func (p GetBatchMarketCandlesticksParams) toMap() map[string]string {
	return NewQuery().
		String("market_tickers", p.MarketTickers).
		Int64("start_ts", p.StartTs).
		Int64("end_ts", p.EndTs).
		Int("period_interval", p.PeriodInterval).
		Bool("include_latest_before_start", p.IncludeLatestBeforeStart).
		Build()
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// OrderbookFP is the fixed-point orderbook from GET /markets/{ticker}/orderbook.
type OrderbookFP struct {
	YesDollars [][]string `json:"yes_dollars"`
	NoDollars  [][]string `json:"no_dollars"`
}

// GetMarketOrderbookResponse is the response from GET /markets/{ticker}/orderbook.
type GetMarketOrderbookResponse struct {
	OrderbookFP OrderbookFP `json:"orderbook_fp"`
}

// MarketOrderbookEntry is a single market's orderbook in a batch response.
type MarketOrderbookEntry struct {
	Ticker      string      `json:"ticker"`
	OrderbookFP OrderbookFP `json:"orderbook_fp"`
}

// GetMarketOrderbooksResponse is the response from GET /markets/orderbooks.
type GetMarketOrderbooksResponse struct {
	Orderbooks []MarketOrderbookEntry `json:"orderbooks"`
}

// TradeResponse is a single trade record from the Kalshi API.
type TradeResponse struct {
	TradeID         string `json:"trade_id"`
	Ticker          string `json:"ticker"`
	CountFP         string `json:"count_fp"`
	YesPriceDollars string `json:"yes_price_dollars"`
	NoPriceDollars  string `json:"no_price_dollars"`
	TakerSide       Side   `json:"taker_side"`
	CreatedTime     string `json:"created_time"`
}

// GetTradesResponse is the paginated response from GET /markets/trades.
type GetTradesResponse struct {
	Trades []TradeResponse `json:"trades"`
	Cursor string          `json:"cursor"`
}

// CandlestickOHLC is OHLC data for a candlestick period.
type CandlestickOHLC struct {
	OpenDollars  string `json:"open_dollars"`
	LowDollars   string `json:"low_dollars"`
	HighDollars  string `json:"high_dollars"`
	CloseDollars string `json:"close_dollars"`
}

// CandlestickPriceOHLC is extended OHLC data for traded prices in a candlestick period.
type CandlestickPriceOHLC struct {
	OpenDollars     string `json:"open_dollars"`
	LowDollars      string `json:"low_dollars"`
	HighDollars     string `json:"high_dollars"`
	CloseDollars    string `json:"close_dollars"`
	MeanDollars     string `json:"mean_dollars"`
	PreviousDollars string `json:"previous_dollars"`
	MinDollars      string `json:"min_dollars"`
	MaxDollars      string `json:"max_dollars"`
}

// Candlestick is a single candlestick data point.
type Candlestick struct {
	EndPeriodTs    int64                `json:"end_period_ts"`
	YesBid         CandlestickOHLC      `json:"yes_bid"`
	YesAsk         CandlestickOHLC      `json:"yes_ask"`
	Price          CandlestickPriceOHLC `json:"price"`
	VolumeFP       string               `json:"volume_fp"`
	OpenInterestFP string               `json:"open_interest_fp"`
}

// GetMarketCandlesticksResponse is the response from GET /series/{series}/markets/{ticker}/candlesticks.
type GetMarketCandlesticksResponse struct {
	Ticker       string        `json:"ticker"`
	Candlesticks []Candlestick `json:"candlesticks"`
}

// MarketCandlesticks groups candlestick data for a single market in a batch response.
type MarketCandlesticks struct {
	MarketTicker string        `json:"market_ticker"`
	Candlesticks []Candlestick `json:"candlesticks"`
}

// GetBatchMarketCandlesticksResponse is the response from GET /markets/candlesticks.
type GetBatchMarketCandlesticksResponse struct {
	Markets []MarketCandlesticks `json:"markets"`
}

// PriceRange defines a valid price range for orders on a market.
type PriceRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Step  string `json:"step"`
}

// MVESelectedLeg is a leg in a multivariate event market.
type MVESelectedLeg struct {
	EventTicker               string `json:"event_ticker"`
	MarketTicker              string `json:"market_ticker"`
	Side                      Side   `json:"side"`
	YesSettlementValueDollars string `json:"yes_settlement_value_dollars"`
}

// MarketDetail is the full market object returned by the Kalshi API.
type MarketDetail struct {
	Ticker      string     `json:"ticker"`
	EventTicker string     `json:"event_ticker"`
	MarketType  MarketType `json:"market_type"`
	YesSubTitle string     `json:"yes_sub_title"`
	NoSubTitle  string     `json:"no_sub_title"`

	CreatedTime             string `json:"created_time"`
	UpdatedTime             string `json:"updated_time"`
	OpenTime                string `json:"open_time"`
	CloseTime               string `json:"close_time"`
	LatestExpirationTime    string `json:"latest_expiration_time"`
	SettlementTimerSeconds  int    `json:"settlement_timer_seconds"`
	ExpectedExpirationTime  string `json:"expected_expiration_time"`
	SettlementTs            string `json:"settlement_ts"`
	FeeWaiverExpirationTime string `json:"fee_waiver_expiration_time"`

	Status                   MarketStatus `json:"status"`
	Result                   MarketResult `json:"result"`
	CanCloseEarly            bool         `json:"can_close_early"`
	FractionalTradingEnabled bool         `json:"fractional_trading_enabled"`
	IsProvisional            bool         `json:"is_provisional"`

	YesBidDollars         string `json:"yes_bid_dollars"`
	YesBidSizeFP          string `json:"yes_bid_size_fp"`
	YesAskDollars         string `json:"yes_ask_dollars"`
	YesAskSizeFP          string `json:"yes_ask_size_fp"`
	NoBidDollars          string `json:"no_bid_dollars"`
	NoAskDollars          string `json:"no_ask_dollars"`
	LastPriceDollars      string `json:"last_price_dollars"`
	PreviousYesBidDollars string `json:"previous_yes_bid_dollars"`
	PreviousYesAskDollars string `json:"previous_yes_ask_dollars"`
	PreviousPriceDollars  string `json:"previous_price_dollars"`

	VolumeFP             string `json:"volume_fp"`
	Volume24hFP          string `json:"volume_24h_fp"`
	OpenInterestFP       string `json:"open_interest_fp"`
	NotionalValueDollars string `json:"notional_value_dollars"`

	RulesPrimary        string       `json:"rules_primary"`
	RulesSecondary      string       `json:"rules_secondary"`
	PriceLevelStructure string       `json:"price_level_structure"`
	PriceRanges         []PriceRange `json:"price_ranges"`
	EarlyCloseCondition string       `json:"early_close_condition"`

	ExpirationValue        string `json:"expiration_value"`
	SettlementValueDollars string `json:"settlement_value_dollars"`

	StrikeType       string   `json:"strike_type"`
	FloorStrike      *float64 `json:"floor_strike"`
	CapStrike        *float64 `json:"cap_strike"`
	FunctionalStrike string   `json:"functional_strike"`
	CustomStrike     any      `json:"custom_strike"`

	MVECollectionTicker string           `json:"mve_collection_ticker"`
	MVESelectedLegs     []MVESelectedLeg `json:"mve_selected_legs"`

	Title              string `json:"title"`
	Subtitle           string `json:"subtitle"`
	LiquidityDollars   string `json:"liquidity_dollars"`
	ExpirationTime     string `json:"expiration_time"`
	ResponsePriceUnits string `json:"response_price_units"`
	TickSize           int    `json:"tick_size"`

	PrimaryParticipantKey string `json:"primary_participant_key"`
}

// MarketResponse is the response from GET /markets/{ticker}.
type MarketResponse struct {
	Market MarketDetail `json:"market"`
}

// GetMarketsResponse is the paginated response from GET /markets.
type GetMarketsResponse struct {
	Markets []MarketDetail `json:"markets"`
	Cursor  string         `json:"cursor"`
}
