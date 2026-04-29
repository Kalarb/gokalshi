![Unit Tests](https://github.com/Kalarb/gokalshi/actions/workflows/unit-tests.yml/badge.svg)
![OpenAPI Validation](https://github.com/Kalarb/gokalshi/actions/workflows/openapi-validation.yml/badge.svg)
![AsyncAPI Validation](https://github.com/Kalarb/gokalshi/actions/workflows/asyncapi-validation.yml/badge.svg)
![HTTP Integration](https://github.com/Kalarb/gokalshi/actions/workflows/http-integration.yml/badge.svg)
![WS Integration](https://github.com/Kalarb/gokalshi/actions/workflows/ws-integration.yml/badge.svg)

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

## API Coverage

### HTTP (37 methods)

| Domain | Methods |
|---|---|
| Account | `GetAccountAPILimits` |
| Exchange | `GetExchangeStatus`, `GetExchangeAnnouncements`, `GetExchangeSchedule`, `GetUserDataTimestamp`, `GetSeriesFeeChanges` |
| Orders | `CreateOrder`, `CancelOrder`, `GetOrder`, `GetOrders`, `BatchCreateOrders`, `BatchCancelOrders`, `AmendOrder`, `DecreaseOrder`, `GetQueuePositions`, `GetQueuePosition` |
| Portfolio | `GetBalance`, `GetPositions`, `GetFills`, `GetSettlements` |
| Markets | `GetMarketOrderbook`, `GetMarketOrderbooks`, `GetTrades`, `GetMarket`, `GetMarkets`, `GetMarketCandlesticks`, `GetBatchMarketCandlesticks` |
| Events | `GetEvent`, `GetEvents`, `GetEventMetadata`, `GetMultivariateEvents`, `GetEventCandlesticks`, `GetEventForecastPercentileHistory` |
| Series | `GetSeries`, `GetSeriesList` |
| Search | `GetTagsByCategories`, `GetFiltersBySport` |

### WebSocket

| Method | Description |
|---|---|
| `Connect` | Establish authenticated WS connection |
| `Close` | Graceful close |
| `ListenLoop` | Read loop with auto-reconnect |
| `MsgCh` | Channel for incoming messages |
| `AddMarkets` | Subscribe tickers to channels |
| `RemoveMarkets` | Unsubscribe tickers from channels |

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
  auth.go              Credentials (RSA-PSS signing)
  config.go            Environment + ClientConfig (env vars)
  errors.go            Typed error hierarchy
  logger.go            slog discard handler + WithWSLogger option
  interfaces.go        HTTPClient + WebSocketClient interfaces

  http_client.go       Client (rate limiting, retry, auth)
  ws_client.go         WSClient (subscriptions, reconnect, slog)

  enums.go             Typed string enums
  query.go             QueryBuilder for query params
  ratelimit.go         ReadWriteTokenBucket (disjoint read/write)

  ws_types.go          ChannelState, WSMessage, MsgTypeToChannel
  ws_messages.go       WS command/response/data message types

  api_*.go             One file per API domain (37 methods total)
  *_params.go          Query parameter structs
  *_responses.go       Response type structs
  *_requests.go        Request body structs
```
