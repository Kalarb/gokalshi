// Copyright 2026 Kalarb
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
