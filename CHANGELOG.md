# Changelog

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
