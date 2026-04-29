package gokalshi

// Response types for market-related API endpoints.

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
	Ticker      string     `json:"ticker"`
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
