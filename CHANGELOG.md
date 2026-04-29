# Changelog

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
