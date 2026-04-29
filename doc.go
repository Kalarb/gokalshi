// Package gokalshi is a Go SDK for the Kalshi prediction market API.
//
// Pure 1:1 reflection of the Kalshi API. Each method maps to exactly one
// API endpoint. Pagination loops, delta polling, and other application-level
// helpers belong in the consumer, not here.
//
// # HTTP Client
//
// Create a client from environment variables:
//
//	cfg, err := gokalshi.NewClientConfig()
//	client := gokalshi.NewClient(cfg)
//	defer client.Close()
//
//	resp, err := client.GetMarkets(ctx, gokalshi.GetMarketsParams{Status: "open", Limit: 10})
//
// # WebSocket Client
//
// Subscribe to real-time market data:
//
//	ws := gokalshi.NewWSClient(cfg, gokalshi.WithWSLogger(slog.Default()))
//	go ws.ListenLoop(ctx)
//	ws.AddMarkets(ctx, []string{"KXBTC-100K"}, []string{"orderbook_delta", "ticker"})
//
//	for raw := range ws.MsgCh() {
//	    // process message
//	}
//
// # Typed Errors
//
// All errors support errors.As for type assertion:
//
//   - [APIError] for non-429 HTTP errors
//   - [RateLimitError] when 429 retries are exhausted
//   - [AuthError] for RSA key/signing failures
//   - [WebSocketError] for WS connection issues
//   - [SequenceGapError] for WS sequence gaps
//
// # Interfaces
//
// Consumers should type-annotate against [HTTPClient] and [WebSocketClient]
// interfaces for testability and loose coupling.
package gokalshi
