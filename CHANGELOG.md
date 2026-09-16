# Changelog

## Unreleased

### Fixed

- **The rate limiter mis-billed most endpoints it was configured for.**
  `GET /account/endpoint_costs` returns paths with Gin-style placeholders
  (`/portfolio/events/orders/:order_id`), but the pattern builder only
  recognised OpenAPI's `{param}`. Every parameterized entry compiled to a
  literal regex that could never match a request, so it fell through to the
  default cost — silently, since falling back is indistinguishable from an
  endpoint that has no override. **Nine of the seventeen live entries were
  dead**, including order cancellation, which billed 10 tokens against a real
  cost of 2. Matching is now segment-wise and recognises every placeholder
  syntax in circulation; all seventeen live entries resolve.
- **Batch endpoints were billed as a single request.** `resolveCosts` replaced
  the caller's figure with a flat per-request cost, discarding the per-item
  multiplication a batch call site had already done. The cost table cannot
  express a multiplier — `EndpointTokenCost` is a flat integer — so the count
  has to come from the caller. A fifty-order batch create was billed 10 tokens
  against a real cost of 500. Note `POST /portfolio/events/orders/batched` is
  absent from the table entirely, because its per-item cost equals the default,
  so the multiplier is the only thing that can make it correct.
- `BatchCancelOrdersV2` used 10 tokens per order where the documented and
  observed cost is 2, and `CancelOrderV2` used 1 where it is 2. These apply on
  the degraded path, when auto-configuration fails — which is also when the
  limiter falls back to Basic-tier bucket sizes, so accuracy matters most.
- `ConfigureRateLimits` now warns when a cost entry contains a segment that
  looks like an unrecognised path parameter, so a future syntax change surfaces
  as a log line rather than as quietly mis-billed endpoints.

### Changed

- Cost tests now drive a captured live `endpoint_costs` payload through
  `ConfigureRateLimits`. The previous tests hand-built patterns with
  placeholders already substituted, so they exercised the matcher and never the
  construction — which is where the defect was, and why it survived.

## v1.0.0 — unreleased

Realigns the SDK with the Kalshi API as of OpenAPI 3.30.0. All 109 HTTP
endpoints and all 13 WebSocket channels are implemented, and the spec drift
tests run with an empty skip list.

### Breaking

- **The legacy V1 order-write methods are removed.** Kalshi withdrew these
  endpoints and now answers them with `410 Gone`, so they had already stopped
  working:

  | Removed | Replacement |
  |---|---|
  | `CreateOrder` | `CreateOrderV2` |
  | `CancelOrder` | `CancelOrderV2` |
  | `AmendOrder` | `AmendOrderV2` |
  | `DecreaseOrder` | `DecreaseOrderV2` |
  | `BatchCreateOrders` | `BatchCreateOrdersV2` |
  | `BatchCancelOrders` | `BatchCancelOrdersV2` |

  The V2 request shape uses the single-book model: `Side` is `BookSideBid` or
  `BookSideAsk` on the YES leg rather than a `Side`/`Action` pair, and `Count`
  and `Price` are fixed-point decimal strings (`"10.00"`, `"0.5600"`).

  Order *reads* are unaffected: `GetOrder`, `GetOrders` and the queue-position
  methods stay on the V1 paths, which Kalshi still serves.

- `GetExchangeAnnouncements` removed — endpoint discontinued 2026-07-04.
- `GetMultivariateEventCollectionLookupHistory` and
  `LookupTickersForMarketInMultivariateEventCollection` removed — endpoints
  removed 2026-08-06.
- `WSMsgMultivariateLookup` and `MultivariateLookupData` removed — the
  `multivariate` channel left the AsyncAPI spec on 2026-08-06.
- `GetOrder` returns `GetOrderResponse` instead of `CreateOrderResponse`.
- `ApiKey.Scopes` is `[]ApiKeyScope` instead of `[]string`.
- `CreateOrderV2Request.TimeInForce` is `TimeInForce` instead of `string`.
- `CancelOrderV2Params.ExchangeIndex` is `*int` instead of `int`. `0` selects
  the event-contract instance and `-1` auto-routes by market ticker; as a plain
  int both were indistinguishable from unset and were silently dropped.

### Added

- 20 endpoints: `CancelAllOrders`, weather index (2), block trade proposals
  (3), target balance allocation (2), intra-exchange transfers (3), FCM (2),
  RFQ-scoped quote operations (4), `GetHistoricalPositions`,
  `GetEventLiveData`, `GetAccountAPIUsageLevelVolumeProgress`.
- WebSocket channels `cfbenchmarks_value_5hz` and `pyth_value`, with their four
  message types and pyth's own update actions.
- Enum types `ApiKeyScope`, `ExchangeInstance`, `RestingMarginReservation`,
  `IntraExchangeInstanceTransferStatus`, `UserFilter`, and the
  `quadratic_with_combo_maker_fees` fee type.
- `exchange_index` across the schemas Kalshi added it to, plus the weather,
  block-trade, target-balance and API-usage-level types.
- `QueryBuilder.IntPtr` for parameters whose meaningful values include 0 and -1.
- FCM endpoints are implemented but held out of the drift comparison: they
  return 403 without FCM membership, so their contract cannot be exercised.
  The skip-list entry records that reason.
- `specs/` — a pinned spec snapshot with version and sha256. Generators and
  drift tests read it instead of fetching docs.kalshi.com at build time, so
  regeneration is reproducible and a release can be traced to an exact spec.
  `tools/vendor_spec.sh` refreshes it; `--check` reports drift against live.

### Fixed

- **The WebSocket base URLs were wrong, and no WS connection could succeed.**
  `prodWSBase` and `demoWSBase` pointed at `external-api`, but Kalshi serves
  WebSockets from a separate `external-api-ws` host — the old host answers 404
  on `/trade-api/ws/v2`, the correct one answers 401. All 16 WebSocket
  integration tests failed to connect; all 16 pass now. Added
  `TestAsyncAPIServerURLs` and `TestOpenAPIServerURLs`, which compare the SDK's
  base URLs against the servers the specs declare — the base URLs were the one
  hand-written thing left in an otherwise spec-driven client, so nothing was
  watching them.

- The drift tests only ever checked for *missing* endpoints, so an endpoint
  Kalshi had withdrawn could ship indefinitely. They now also fail on endpoints,
  WS channels and message types that are absent from the spec.
- Added `TestOpenAPISchemaCoverage`: Go had no REST schema drift check at all,
  so new fields surfaced only as runtime deserialization failures.
- `tools/generate_types` discarded enum schemas on the stale assumption that
  `enums.go` held a superset, which is how two referenced enum types came to be
  missing entirely.
- `tools/generate_ws_types` built error-code constant names from the spec's
  human-readable message text, emitting invalid Go for codes 27 and 28.
- `tools/generate_coverage` counted only tests named after their method, so
  table-driven tests read as zero coverage. It now also counts call sites.
- Deprecated the quote-id-only RFQ operations in favour of the RFQ-scoped forms,
  matching the spec.

## v0.2.0 — 2026-05-28

Full Kalshi API parity. 97 of 99 OpenAPI endpoints implemented (2 FCM-only endpoints intentionally skipped).

### Added
- 60 new HTTP API methods across 12 new domain files:
  - `api_keys.go` — GetAPIKeys, CreateAPIKey, GenerateAPIKey, DeleteAPIKey
  - `communications.go` — GetCommunicationsID, RFQ CRUD (5), Quote CRUD + accept/confirm (6)
  - `event_orders.go` — V2 order endpoints: Create, Cancel, Amend, Decrease, BatchCreate, BatchCancel
  - `historical.go` — GetHistoricalCutoff, Markets, Market, MarketCandlesticks, Fills, Orders, Trades
  - `incentive.go` — GetIncentivePrograms
  - `live_data.go` — GetLiveData, GetLiveDataLegacy, GetLiveDataBatch, GetGameStats
  - `milestones.go` — GetMilestones, GetMilestone
  - `mve_collections.go` — CRUD + lookup for multivariate event collections (5)
  - `order_groups.go` — Create, Get, Delete, Reset, Trigger, UpdateLimit (7)
  - `structured_targets.go` — GetStructuredTargets, GetStructuredTarget
  - `subaccounts.go` — Create, GetBalances, GetNetting, UpdateNetting, Transfer, GetTransfers
  - `summary.go` — GetPortfolioRestingOrderTotalValue
- 3 new methods in existing domain files: GetEndpointCosts, GetDeposits, GetWithdrawals
- `put()` and `putJSON()` HTTP verb support in client
- `LoadCredentialsFromPEM` — parse RSA key from PEM string (useful for CI env vars)
- `NewCredentials` — create credentials from a pre-loaded RSA private key
- `GetEventFeeUpdates` in events.go
- Unit tests for all new endpoints (248+ total)
- Version bump & release CI workflow (automated tagging on conventional commits)

### Changed
- HTTPClient interface expanded from 37 to 97 methods
- types_generated.go regenerated with 40+ new request/response types
- README updated with full 97-method API coverage tables

## v0.1.1 — 2026-05-28

Open source release.

### Changed
- License from proprietary to Apache 2.0
- Consolidated 5 CI workflows (lint, rest-unit, ws-unit, http-integration, ws-integration) into single `ci.yml`
- OpenAPI drift test now strictly detects missing endpoints (was silently passing with soft thresholds)

### Added
- CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md
- Dependabot for daily gomod and github-actions updates
- Apache 2.0 header in doc.go
- Clickable badge links in README

### Fixed

- **The WebSocket base URLs were wrong, and no WS connection could succeed.**
  `prodWSBase` and `demoWSBase` pointed at `external-api`, but Kalshi serves
  WebSockets from a separate `external-api-ws` host — the old host answers 404
  on `/trade-api/ws/v2`, the correct one answers 401. All 16 WebSocket
  integration tests failed to connect; all 16 pass now. Added
  `TestAsyncAPIServerURLs` and `TestOpenAPIServerURLs`, which compare the SDK's
  base URLs against the servers the specs declare — the base URLs were the one
  hand-written thing left in an otherwise spec-driven client, so nothing was
  watching them.
- LICENSE formatting for pkg.go.dev detection

## v0.1.0 — 2026-04-28

Initial release. Extracted from `trading-system/internal/kalshi/`.

### Added
- HTTP client with RSA-PSS auth, rate limiting, and 429 retry
- WebSocket client with auto-reconnect, subscription management, and slog logging
- 37 HTTP API methods: account, exchange, orders, portfolio, markets, events, series, search
- 17 WebSocket message types across 11 channels
- Typed error hierarchy: APIError, RateLimitError, AuthError, WebSocketError, SequenceGapError
- HTTPClient and WebSocketClient interfaces with compile-time checks
- Unit tests, spec validation tests (OpenAPI + AsyncAPI), and integration tests (HTTP + WS)
