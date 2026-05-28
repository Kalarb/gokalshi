# API Coverage

gokalshi implements **97 of 99** Kalshi OpenAPI endpoints (2 FCM-only endpoints intentionally skipped) and all **11 AsyncAPI WebSocket channels**.

Coverage is validated automatically against the live OpenAPI and AsyncAPI specs via CI ([OpenAPI workflow](https://github.com/Kalarb/gokalshi/actions/workflows/openapi-validation.yml), [AsyncAPI workflow](https://github.com/Kalarb/gokalshi/actions/workflows/asyncapi-validation.yml)).

## HTTP Endpoints (97)

| Domain | File | Endpoints | Unit | Integration |
|--------|------|:---------:|:----:|:-----------:|
| Account | `account.go` | 2 | 2 | 1 |
| Exchange | `exchange.go` | 5 | 5 | 5 |
| Orders | `orders.go` | 10 | 10 | 10 |
| Event Orders (V2) | `event_orders.go` | 6 | 6 | 0 |
| Portfolio | `portfolio.go` | 7 | 7 | 4 |
| Subaccounts | `subaccounts.go` | 6 | 6 | 0 |
| Order Groups | `order_groups.go` | 7 | 7 | 0 |
| Markets | `markets.go` | 7 | 7 | 7 |
| Events | `events.go` | 7 | 7 | 6 |
| Series | `series.go` | 2 | 2 | 2 |
| Search | `search.go` | 2 | 2 | 2 |
| Communications | `communications.go` | 11 | 11 | 0 |
| API Keys | `api_keys.go` | 4 | 4 | 0 |
| Historical | `historical.go` | 7 | 7 | 0 |
| Incentive Programs | `incentive.go` | 1 | 1 | 0 |
| Live Data | `live_data.go` | 4 | 4 | 0 |
| Milestones | `milestones.go` | 2 | 2 | 0 |
| Multivariate Collections | `mve_collections.go` | 5 | 5 | 0 |
| Structured Targets | `structured_targets.go` | 2 | 2 | 0 |
| Summary | `summary.go` | 1 | 1 | 0 |
| **Total** | | **97** | **97** | **37** |

### Skipped Endpoints (2)

| Endpoint | Reason |
|----------|--------|
| `/fcm/rate-limits` | FCM-only, not available via standard API keys |
| `/fcm/live` | FCM-only, not available via standard API keys |

### Integration Test Notes

Integration tests run against the live Kalshi DEMO API. Endpoints without integration tests fall into categories:
- **Not available on DEMO** (communications, order groups, subaccounts, API keys)
- **Require specific data state** (historical, live data, milestones, structured targets)
- **V2 endpoints** (event orders — DEMO may not support single-book model)

## WebSocket Channels (11)

| Channel | Auth | Message Types | Unit | Integration |
|---------|:----:|---------------|:----:|:-----------:|
| `orderbook_delta` | | `orderbook_snapshot`, `orderbook_delta` | Y | Y |
| `ticker` | | `ticker` | Y | Y |
| `trade` | | `trade` | Y | Y |
| `fill` | Y | `fill` | Y | Y |
| `market_positions` | Y | `market_position` | Y | Y |
| `user_orders` | Y | `user_order` | Y | Y |
| `market_lifecycle_v2` | | `market_lifecycle_v2`, `event_lifecycle`, `event_fee_update` | Y | Y |
| `multivariate_market_lifecycle` | | `multivariate_market_lifecycle`, `event_lifecycle` | Y | Y |
| `multivariate` | | `multivariate_lookup` | Y | Y |
| `communications` | Y | `rfq_created`, `rfq_deleted`, `quote_created`, `quote_accepted`, `quote_executed` | Y | Y |
| `order_group_updates` | Y | `order_group_updates` | Y | Y |
