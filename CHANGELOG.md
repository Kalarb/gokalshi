# Changelog

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
