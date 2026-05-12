![REST Unit Tests](https://github.com/Kalarb/gokalshi/actions/workflows/rest-unit-tests.yml/badge.svg)
![WS Unit Tests](https://github.com/Kalarb/gokalshi/actions/workflows/ws-unit-tests.yml/badge.svg)
![Lint](https://github.com/Kalarb/gokalshi/actions/workflows/lint.yml/badge.svg)
![HTTP Integration](https://github.com/Kalarb/gokalshi/actions/workflows/http-integration.yml/badge.svg)
![WS Integration](https://github.com/Kalarb/gokalshi/actions/workflows/ws-integration.yml/badge.svg)
![OpenAPI](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/Kalarb/65579a629076066fcbf09520ca76301a/raw/go-openapi-status.json)
![AsyncAPI](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/Kalarb/65579a629076066fcbf09520ca76301a/raw/go-asyncapi-status.json)

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

### HTTP (37 methods)

#### Account

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetAccountAPILimits` | Y | Y | Y | |

#### Exchange

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetExchangeStatus` | Y | Y | Y | |
| `GetExchangeAnnouncements` | Y | Y | Y | |
| `GetExchangeSchedule` | Y | Y | Y | |
| `GetUserDataTimestamp` | Y | Y | Y | |
| `GetSeriesFeeChanges` | Y | Y | Y | |

#### Orders

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetOrders` | Y | Y | Y | |
| `GetOrder` | Y | Y | Y | retry for propagation delay |
| `CreateOrder` | Y | Y | Y | places at 1c, won't fill |
| `CancelOrder` | Y | Y | Y | cleans up created order |
| `AmendOrder` | Y | Y | Y | create -> amend price -> cancel |
| `DecreaseOrder` | Y | Y | Y | create count=2 -> decrease to 1 -> cancel |
| `BatchCreateOrders` | Y | Y | Y | batch create 3 -> batch cancel all |
| `BatchCancelOrders` | Y | Y | Y | same test as batch_create |
| `GetQueuePositions` | Y | Y | Y | filters by market ticker |
| `GetQueuePosition` | Y | Y | Y | skips if 404 on DEMO |

#### Portfolio

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetBalance` | Y | Y | Y | |
| `GetPositions` | Y | Y | Y | |
| `GetFills` | Y | Y | Y | |
| `GetSettlements` | Y | Y | Y | |

#### Markets

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetMarket` | Y | Y | Y | |
| `GetMarkets` | Y | Y | Y | |
| `GetMarketOrderbook` | Y | Y | Y | |
| `GetMarketOrderbooks` | Y | Y | Y | batch, multiple tickers |
| `GetTrades` | Y | Y | Y | |
| `GetMarketCandlesticks` | Y | Y | Y | via series + market ticker |
| `GetBatchMarketCandlesticks` | Y | Y | Y | |

#### Events

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetEvent` | Y | Y | Y | |
| `GetEvents` | Y | Y | Y | |
| `GetEventMetadata` | Y | Y | Y | |
| `GetMultivariateEvents` | Y | Y | Y | |
| `GetEventCandlesticks` | Y | Y | Y | via series + event ticker |
| `GetEventForecastPercentileHistory` | Y | Y | Y | skips if 400 on DEMO |

#### Series

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetSeries` | Y | Y | Y | |
| `GetSeriesList` | Y | Y | Y | |

#### Search

| Method | Impl | Unit | Integration | Notes |
|---|:---:|:---:|:---:|---|
| `GetTagsByCategories` | Y | Y | Y | |
| `GetFiltersBySport` | Y | Y | Y | |

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
| `market_lifecycle_v2` | | `market_lifecycle_v2`, `event_lifecycle` | Y | Y | Y | subscription-only in integration |
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
    account.go         Account API limits
    events.go          Events, metadata, candlesticks, forecasts
    exchange.go        Exchange status, schedule, announcements
    markets.go         Markets, orderbooks, trades, candlesticks
    orders.go          Orders CRUD, batch, amend, decrease, queue
    portfolio.go       Balance, positions, fills, settlements
    search.go          Tags, filters
    series.go          Series, fee changes

  WebSocket:
    ws_types.go        ChannelState, WSMessage, MsgTypeToChannel
    ws_messages.go     WS command/response/data message types
```
