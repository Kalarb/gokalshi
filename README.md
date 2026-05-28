[![Go Reference](https://pkg.go.dev/badge/github.com/Kalarb/gokalshi.svg)](https://pkg.go.dev/github.com/Kalarb/gokalshi)
[![CI](https://github.com/Kalarb/gokalshi/actions/workflows/ci.yml/badge.svg)](https://github.com/Kalarb/gokalshi/actions/workflows/ci.yml)
[![OpenAPI](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/yardboy27/65579a629076066fcbf09520ca76301a/raw/go-openapi-status.json)](https://github.com/Kalarb/gokalshi/actions/workflows/openapi-validation.yml)
[![AsyncAPI](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/yardboy27/65579a629076066fcbf09520ca76301a/raw/go-asyncapi-status.json)](https://github.com/Kalarb/gokalshi/actions/workflows/asyncapi-validation.yml)

# gokalshi

Go SDK for the [Kalshi](https://kalshi.com) prediction market API.

Pure 1:1 reflection of the Kalshi API. Each method maps to exactly one API endpoint. Pagination loops, delta polling, and other application-level helpers belong in the consumer, not here.

## Install

```bash
go get github.com/Kalarb/gokalshi
```

## Quick Start

### HTTP Client

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Kalarb/gokalshi"
)

func main() {
    cfg, err := gokalshi.NewClientConfig() // reads env vars
    if err != nil {
        log.Fatal(err)
    }

    client := gokalshi.NewClient(cfg)
    defer client.Close()

    ctx := context.Background()

    // Markets
    resp, err := client.GetMarkets(ctx, gokalshi.GetMarketsParams{Status: "open", Limit: 10})
    if err != nil {
        log.Fatal(err)
    }
    for _, m := range resp.Markets {
        fmt.Printf("%s: %s/%s\n", m.Ticker, m.YesBidDollars, m.YesAskDollars)
    }

    // Place an order
    result, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
        Ticker:          "KXBTC-100K",
        Side:            gokalshi.SideYes,
        Action:          gokalshi.ActionBuy,
        CountFP:         "10.00",
        YesPriceDollars: "0.5000",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Order.OrderID)
}
```

### WebSocket Client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "log/slog"
    "os"

    "github.com/Kalarb/gokalshi"
)

func main() {
    cfg, err := gokalshi.NewClientConfig()
    if err != nil {
        log.Fatal(err)
    }

    logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
    ws := gokalshi.NewWSClient(cfg, gokalshi.WithWSLogger(logger))

    ctx := context.Background()
    go ws.ListenLoop(ctx)

    if err := ws.AddMarkets(ctx, []string{"KXBTC-100K"}, []string{"orderbook_delta", "ticker"}); err != nil {
        log.Fatal(err)
    }

    for raw := range ws.MsgCh() {
        var msg gokalshi.WSMessage
        json.Unmarshal(raw, &msg)
        fmt.Printf("type=%s sid=%d seq=%d\n", msg.Type, msg.SID, msg.Seq)
    }
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `KALSHI_ENVIRONMENT` | `DEMO` | `DEMO` or `PROD` |
| `KALSHI_DEMO_API_KEY_ID` | | API key ID for demo |
| `KALSHI_DEMO_PRIVATE_KEY_FILE` | | Path to PEM private key for demo |
| `KALSHI_PROD_API_KEY_ID` | | API key ID for production |
| `KALSHI_PROD_PRIVATE_KEY_FILE` | | Path to PEM private key for production |
| `KALSHI_HTTP_BASE_URL` | Derived from environment | Override HTTP base URL |
| `KALSHI_WS_BASE_URL` | Derived from environment | Override WS base URL |

### Client Options

| Option | Default | Description |
|---|---|---|
| `WithHTTPClient(*http.Client)` | 30s timeout | Custom HTTP client |
| `WithMaxRetries(n)` | 4 | Max 429 retry count |
| `WithBaseDelay(d)` | 100ms | Initial backoff delay |
| `WithRateLimiter(l)` | 20r/10w per sec | Custom rate limiter |

### WebSocket Options

| Option | Default | Description |
|---|---|---|
| `WithWSMsgBufSize(n)` | 4096 | Message channel buffer |
| `WithWSBackoff(min, max)` | 1s / 32s | Reconnection backoff |
| `WithWSLogger(*slog.Logger)` | discard | Structured logger for lifecycle events |

## API Coverage & Tests

Every implemented endpoint has a unit test (mock HTTP server). Integration tests hit the real Kalshi DEMO API (HTTP) or PROD API (WebSocket, read-only).

### HTTP (97 methods)

#### Account

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetAccountAPILimits` | Y | Y | Y |
| `GetAccountEndpointCosts` | Y | Y | |

#### Exchange

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetExchangeStatus` | Y | Y | Y |
| `GetExchangeAnnouncements` | Y | Y | Y |
| `GetExchangeSchedule` | Y | Y | Y |
| `GetUserDataTimestamp` | Y | Y | Y |
| `GetSeriesFeeChanges` | Y | Y | Y |

#### Orders

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetOrders` | Y | Y | Y |
| `GetOrder` | Y | Y | Y |
| `CreateOrder` | Y | Y | Y |
| `CancelOrder` | Y | Y | Y |
| `AmendOrder` | Y | Y | Y |
| `DecreaseOrder` | Y | Y | Y |
| `BatchCreateOrders` | Y | Y | Y |
| `BatchCancelOrders` | Y | Y | Y |
| `GetQueuePositions` | Y | Y | Y |
| `GetQueuePosition` | Y | Y | Y |

#### Event Orders (V2)

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `CreateOrderV2` | Y | Y | |
| `BatchCreateOrdersV2` | Y | Y | |
| `BatchCancelOrdersV2` | Y | Y | |
| `CancelOrderV2` | Y | Y | |
| `AmendOrderV2` | Y | Y | |
| `DecreaseOrderV2` | Y | Y | |

#### Portfolio

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetBalance` | Y | Y | Y |
| `GetPositions` | Y | Y | Y |
| `GetFills` | Y | Y | Y |
| `GetSettlements` | Y | Y | Y |
| `GetDeposits` | Y | Y | |
| `GetWithdrawals` | Y | Y | |
| `GetPortfolioRestingOrderTotalValue` | Y | Y | |

#### Subaccounts

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `CreateSubaccount` | Y | Y | |
| `GetSubaccountBalances` | Y | Y | |
| `GetSubaccountNetting` | Y | Y | |
| `UpdateSubaccountNetting` | Y | Y | |
| `ApplySubaccountTransfer` | Y | Y | |
| `GetSubaccountTransfers` | Y | Y | |

#### Order Groups

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `CreateOrderGroup` | Y | Y | |
| `GetOrderGroups` | Y | Y | |
| `GetOrderGroup` | Y | Y | |
| `DeleteOrderGroup` | Y | Y | |
| `ResetOrderGroup` | Y | Y | |
| `TriggerOrderGroup` | Y | Y | |
| `UpdateOrderGroupLimit` | Y | Y | |

#### Markets

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetMarket` | Y | Y | Y |
| `GetMarkets` | Y | Y | Y |
| `GetMarketOrderbook` | Y | Y | Y |
| `GetMarketOrderbooks` | Y | Y | Y |
| `GetTrades` | Y | Y | Y |
| `GetMarketCandlesticks` | Y | Y | Y |
| `GetBatchMarketCandlesticks` | Y | Y | Y |

#### Events

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetEvent` | Y | Y | Y |
| `GetEvents` | Y | Y | Y |
| `GetEventMetadata` | Y | Y | Y |
| `GetMultivariateEvents` | Y | Y | Y |
| `GetEventCandlesticks` | Y | Y | Y |
| `GetEventForecastPercentileHistory` | Y | Y | Y |
| `GetEventFeeChanges` | Y | Y | |

#### Series

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetSeries` | Y | Y | Y |
| `GetSeriesList` | Y | Y | Y |

#### Search

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetTagsByCategories` | Y | Y | Y |
| `GetFiltersBySport` | Y | Y | Y |

#### Communications (RFQs & Quotes)

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetCommunicationsID` | Y | Y | |
| `CreateRFQ` | Y | Y | |
| `GetRFQs` | Y | Y | |
| `GetRFQ` | Y | Y | |
| `DeleteRFQ` | Y | Y | |
| `CreateQuote` | Y | Y | |
| `GetQuotes` | Y | Y | |
| `GetQuote` | Y | Y | |
| `DeleteQuote` | Y | Y | |
| `AcceptQuote` | Y | Y | |
| `ConfirmQuote` | Y | Y | |

#### API Keys

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetAPIKeys` | Y | Y | |
| `CreateAPIKey` | Y | Y | |
| `GenerateAPIKey` | Y | Y | |
| `DeleteAPIKey` | Y | Y | |

#### Historical

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetHistoricalCutoff` | Y | Y | |
| `GetHistoricalFills` | Y | Y | |
| `GetHistoricalOrders` | Y | Y | |
| `GetHistoricalTrades` | Y | Y | |
| `GetHistoricalMarkets` | Y | Y | |
| `GetHistoricalMarket` | Y | Y | |
| `GetHistoricalMarketCandlesticks` | Y | Y | |

#### Incentive Programs

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetIncentivePrograms` | Y | Y | |

#### Live Data

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetLiveDataBatch` | Y | Y | |
| `GetLiveDataByMilestone` | Y | Y | |
| `GetMilestoneGameStats` | Y | Y | |
| `GetLiveData` | Y | Y | |

#### Milestones

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetMilestones` | Y | Y | |
| `GetMilestone` | Y | Y | |

#### Multivariate Event Collections

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetMultivariateEventCollections` | Y | Y | |
| `GetMultivariateEventCollection` | Y | Y | |
| `GetMultivariateEventCollectionLookupHistory` | Y | Y | |
| `CreateMarketInMultivariateEventCollection` | Y | Y | |
| `LookupTickersForMarketInMultivariateEventCollection` | Y | Y | |

#### Structured Targets

| Method | Impl | Unit | Integration |
|---|:---:|:---:|:---:|
| `GetStructuredTargets` | Y | Y | |
| `GetStructuredTarget` | Y | Y | |

### WebSocket

#### Channels

| Channel | Auth | Message Types | Impl | Unit | Integration | Notes |
|---|:---:|---|:---:|:---:|:---:|---|
| `orderbook_delta` | | `orderbook_snapshot`, `orderbook_delta` | Y | Y | Y | sequence tracking |
| `ticker` | | `ticker` | Y | Y | Y | price/volume/OI updates |
| `trade` | | `trade` | Y | Y | Y | public trade notifications |
| `fill` | Y | `fill` | Y | Y | Y | subscription-only in integration |
| `market_positions` | Y | `market_position` | Y | Y | Y | subscription-only in integration |
| `user_orders` | Y | `user_order` | Y | Y | Y | subscription-only in integration |
| `market_lifecycle_v2` | | `market_lifecycle_v2`, `event_lifecycle`, `event_fee_update` | Y | Y | Y | subscription-only in integration |
| `multivariate_market_lifecycle` | | `multivariate_market_lifecycle`, `event_lifecycle` | Y | Y | Y | subscription-only in integration |
| `multivariate` | | `multivariate_lookup` | Y | Y | Y | subscription-only in integration |
| `communications` | Y | `rfq_created`, `rfq_deleted`, `quote_created`, `quote_accepted`, `quote_executed` | Y | Y | Y | subscription-only in integration |
| `order_group_updates` | Y | `order_group_updates` | Y | Y | Y | subscription-only in integration |

#### Client Operations

| Method | Purpose | Impl | Unit | Integration | Notes |
|---|---|:---:|:---:|:---:|---|
| `Connect` | Establish authenticated WS connection | Y | Y | Y | |
| `Close` | Graceful close | Y | Y | Y | |
| `ListenLoop` | Read loop with auto-reconnect | Y | Y | Y | exponential backoff |
| `MsgCh` | Channel for incoming messages | Y | Y | Y | |
| `AddMarkets` | Subscribe tickers to channels | Y | Y | Y | |
| `RemoveMarkets` | Unsubscribe tickers from channels | Y | Y | Y | |

## Typed Errors

| Error | When |
|---|---|
| `*APIError` | Non-429 HTTP errors (parses JSON code/message) |
| `*RateLimitError` | 429 retries exhausted |
| `*AuthError` | RSA key loading or signing failures |
| `*WebSocketError` | WS connection/protocol errors |
| `*SequenceGapError` | WS message sequence gap |

Use `errors.As` for type assertions:

```go
var apiErr *gokalshi.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("status=%d code=%s\n", apiErr.StatusCode, apiErr.Code)
}
```

## Typed Contract

Consumers should type-annotate against interfaces, not concrete types:

```go
func fetchBalance(client gokalshi.HTTPClient) (int64, error) {
    resp, err := client.GetBalance(context.Background())
    return resp.Balance, err
}

func subscribe(ws gokalshi.WebSocketClient, ticker string) error {
    return ws.AddMarkets(context.Background(), []string{ticker}, []string{"orderbook_delta"})
}
```

## Testing

```bash
# Unit tests
go test ./... -v

# With race detector
go test -race ./... -v

# Spec validation (no credentials needed, fetches live OpenAPI/AsyncAPI specs)
go test -tags spec_validation -v -count=1 -timeout 30s

# Integration tests (requires .env with credentials)
# HTTP tests use DEMO: KALSHI_DEMO_API_KEY_ID, KALSHI_DEMO_PRIVATE_KEY_FILE
# WS tests use PROD read-only: KALSHI_PROD_READ_ONLY_API_KEY_ID, KALSHI_PROD_READ_ONLY_PRIVATE_KEY_FILE
go test -tags integration -v -count=1 -timeout 120s
```

## Architecture

```
gokalshi/
  Core:
    client.go          Client (rate limiting, retry, auth)
    ws_client.go       WSClient (subscriptions, reconnect, slog)
    auth.go            Credentials (RSA-PSS signing)
    config.go          Environment + ClientConfig (env vars)
    errors.go          Typed error hierarchy
    interfaces.go      HTTPClient + WebSocketClient interfaces
    logger.go          slog discard handler
    query.go           QueryBuilder for query params
    ratelimit.go       ReadWriteTokenBucket (disjoint read/write)
    enums.go           Typed string enums

  Domains (methods + params + responses per file):
    account.go         Account API limits, endpoint costs
    api_keys.go        API key management
    communications.go  RFQs, quotes, communications ID
    event_orders.go    V2 event orders (create, batch, amend, decrease)
    events.go          Events, metadata, candlesticks, forecasts, fee changes
    exchange.go        Exchange status, schedule, announcements
    historical.go      Historical data (cutoff, fills, orders, trades, markets)
    incentive.go       Incentive programs
    live_data.go       Live data, game stats
    markets.go         Markets, orderbooks, trades, candlesticks
    milestones.go      Milestones
    mve_collections.go Multivariate event collections
    order_groups.go    Order group lifecycle
    orders.go          Orders CRUD, batch, amend, decrease, queue
    portfolio.go       Balance, positions, fills, settlements, deposits, withdrawals
    search.go          Tags, filters
    series.go          Series, fee changes
    structured_targets.go Structured targets
    subaccounts.go     Subaccount management
    summary.go         Portfolio resting order total value

  WebSocket:
    ws_types.go        ChannelState, WSMessage, MsgTypeToChannel
    ws_messages.go     WS command/response/data message types
```
