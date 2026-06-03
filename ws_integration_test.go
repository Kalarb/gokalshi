//go:build integration

package gokalshi

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// messageCollector reads from MsgCh in a goroutine and stores parsed messages.
type messageCollector struct {
	mu       sync.Mutex
	messages []WSMessage
	raw      [][]byte
}

func newMessageCollector() *messageCollector {
	return &messageCollector{}
}

func (mc *messageCollector) run(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			var msg WSMessage
			if err := json.Unmarshal(data, &msg); err == nil {
				mc.mu.Lock()
				mc.messages = append(mc.messages, msg)
				mc.raw = append(mc.raw, data)
				mc.mu.Unlock()
			}
		}
	}
}

func (mc *messageCollector) ofType(msgType string) []WSMessage {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	var result []WSMessage
	for _, m := range mc.messages {
		if string(m.Type) == msgType {
			result = append(result, m)
		}
	}
	return result
}

func (mc *messageCollector) waitForType(t *testing.T, msgType string, timeout time.Duration) WSMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		matches := mc.ofType(msgType)
		if len(matches) > 0 {
			return matches[0]
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no %s message within %s", msgType, timeout)
	return WSMessage{} // unreachable
}

func (mc *messageCollector) waitForTypeOrSkip(t *testing.T, msgType string, timeout time.Duration) WSMessage {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		matches := mc.ofType(msgType)
		if len(matches) > 0 {
			return matches[0]
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Skipf("no %s message within %s — market may be quiet", msgType, timeout)
	return WSMessage{} // unreachable
}

func (mc *messageCollector) clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = nil
	mc.raw = nil
}

// getActiveBTC15MTicker uses the HTTP client to find an active KXBTC15M market ticker.
func getActiveBTC15MTicker(t *testing.T, c *Client) string {
	t.Helper()
	ctx := context.Background()

	eventsResp, err := c.GetEvents(ctx, GetEventsParams{
		Status:            "open",
		SeriesTicker:      "KXBTC15M",
		WithNestedMarkets: true,
		Limit:             1,
	})
	require.NoError(t, err)
	if len(eventsResp.Events) == 0 || len(eventsResp.Events[0].Markets) == 0 {
		t.Skip("no open KXBTC15M events on PROD")
	}
	ticker := eventsResp.Events[0].Markets[0].Ticker
	t.Logf("using BTC15M ticker: %s", ticker)
	return ticker
}

// getActiveBTCDTicker uses the HTTP client to find an active KXBTCD market ticker.
func getActiveBTCDTicker(t *testing.T, c *Client) string {
	t.Helper()
	ctx := context.Background()

	eventsResp, err := c.GetEvents(ctx, GetEventsParams{
		Status:            "open",
		SeriesTicker:      "KXBTCD",
		WithNestedMarkets: true,
		Limit:             1,
	})
	require.NoError(t, err)
	if len(eventsResp.Events) == 0 || len(eventsResp.Events[0].Markets) == 0 {
		t.Skip("no open KXBTCD events on PROD")
	}
	ticker := eventsResp.Events[0].Markets[0].Ticker
	t.Logf("using BTCD ticker: %s", ticker)
	return ticker
}

// wsTestSetup creates HTTP + WS clients and a message collector with ListenLoop running.
func wsTestSetup(t *testing.T) (*Client, *WSClient, *messageCollector, context.Context, context.CancelFunc) {
	t.Helper()
	httpClient := integrationProdHTTPClient(t)
	wsClient := integrationWSClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)

	collector := newMessageCollector()
	go collector.run(ctx, wsClient.MsgCh())
	go wsClient.ListenLoop(ctx)

	// Give connection time to establish.
	time.Sleep(500 * time.Millisecond)

	return httpClient, wsClient, collector, ctx, cancel
}

// ---------------------------------------------------------------------------
// Data channels — subscribe + receive data
// ---------------------------------------------------------------------------

func TestWSIntegration_Orderbook(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()
	_ = wsClient

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"orderbook_delta"})
	require.NoError(t, err)

	msg := collector.waitForType(t, "orderbook_snapshot", 15*time.Second)
	// Parse the msg body to check market_ticker.
	var body OrderbookSnapshotData
	require.NoError(t, json.Unmarshal(msg.Msg, &body))
	assert.Equal(t, ticker, body.MarketTicker)
	t.Logf("received orderbook_snapshot for %s (sid=%d seq=%d)", body.MarketTicker, msg.SID, msg.Seq)
}

func TestWSIntegration_Ticker(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()
	_ = wsClient

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"ticker"})
	require.NoError(t, err)

	msg := collector.waitForTypeOrSkip(t, "ticker", 60*time.Second)
	var body TickerData
	require.NoError(t, json.Unmarshal(msg.Msg, &body))
	assert.Equal(t, ticker, body.MarketTicker)
	t.Logf("received ticker for %s", body.MarketTicker)
}

func TestWSIntegration_Trade(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()
	_ = wsClient

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"trade"})
	require.NoError(t, err)

	msg := collector.waitForTypeOrSkip(t, "trade", 60*time.Second)
	var body TradeData
	require.NoError(t, json.Unmarshal(msg.Msg, &body))
	assert.Equal(t, ticker, body.MarketTicker)
	t.Logf("received trade for %s", body.MarketTicker)
}

func TestWSIntegration_MultiChannel(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()
	_ = wsClient

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"orderbook_delta", "ticker"})
	require.NoError(t, err)

	collector.waitForType(t, "orderbook_snapshot", 15*time.Second)
	collector.waitForTypeOrSkip(t, "ticker", 30*time.Second)

	assert.GreaterOrEqual(t, len(collector.ofType("orderbook_snapshot")), 1)
	t.Logf("received orderbook_snapshot + ticker for %s", ticker)
}

// ---------------------------------------------------------------------------
// Subscribe-only channels — verify subscription succeeds
// ---------------------------------------------------------------------------

func TestWSIntegration_SubscribeFill(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"fill"})
	require.NoError(t, err)

	// "subscribed" is handled internally — poll for SID assignment.
	waitForSID(t, wsClient, "fill", 10*time.Second)
	t.Logf("fill channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeUserOrders(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"user_orders"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "user_orders", 10*time.Second)
	t.Logf("user_orders channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeMarketPositions(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"market_positions"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "market_positions", 10*time.Second)
	t.Logf("market_positions channel subscribed for %s", ticker)
}

// waitForSID polls until the given channel has a non-nil SID (subscription confirmed).
func waitForSID(t *testing.T, ws *WSClient, channel string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ws.mu.Lock()
		state, ok := ws.channels[channel]
		hasSID := ok && state.SID != nil
		ws.mu.Unlock()
		if hasSID {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("channel %s did not receive SID within %s", channel, timeout)
}

// ---------------------------------------------------------------------------
// Subscribe-only channels — verify subscription succeeds (no data expected)
// ---------------------------------------------------------------------------

func TestWSIntegration_SubscribeMarketLifecycleV2(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"market_lifecycle_v2"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "market_lifecycle_v2", 10*time.Second)
	t.Logf("market_lifecycle_v2 channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeMultivariateMarketLifecycle(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"multivariate_market_lifecycle"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "multivariate_market_lifecycle", 10*time.Second)
	t.Logf("multivariate_market_lifecycle channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeMultivariate(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"multivariate"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "multivariate", 10*time.Second)
	t.Logf("multivariate channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeCommunications(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"communications"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "communications", 10*time.Second)
	t.Logf("communications channel subscribed for %s", ticker)
}

func TestWSIntegration_SubscribeOrderGroupUpdates(t *testing.T) {
	httpClient, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	ticker := getActiveBTC15MTicker(t, httpClient)
	err := wsClient.AddMarkets(ctx, []string{ticker}, []string{"order_group_updates"})
	require.NoError(t, err)

	waitForSID(t, wsClient, "order_group_updates", 10*time.Second)
	t.Logf("order_group_updates channel subscribed for %s", ticker)
}

// ---------------------------------------------------------------------------
// Global (ticker-less) subscriptions
// ---------------------------------------------------------------------------

func TestWSIntegration_SubscribeGlobal_Trade(t *testing.T) {
	_, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()

	err := wsClient.Subscribe(ctx, []string{"trade"}, nil)
	require.NoError(t, err)

	waitForSID(t, wsClient, "trade", 10*time.Second)

	// Trade channel is high-volume on PROD; we should receive data quickly.
	msg := collector.waitForType(t, "trade", 15*time.Second)
	var body TradeData
	require.NoError(t, json.Unmarshal(msg.Msg, &body))
	assert.NotEmpty(t, body.MarketTicker, "global trade should include market_ticker")
	t.Logf("received global trade for %s", body.MarketTicker)
}

func TestWSIntegration_SubscribeGlobal_MarketLifecycleV2(t *testing.T) {
	_, wsClient, _, ctx, cancel := wsTestSetup(t)
	defer cancel()

	err := wsClient.Subscribe(ctx, []string{"market_lifecycle_v2"}, nil)
	require.NoError(t, err)

	waitForSID(t, wsClient, "market_lifecycle_v2", 10*time.Second)
	t.Logf("market_lifecycle_v2 globally subscribed")
}

func TestWSIntegration_Unsubscribe(t *testing.T) {
	_, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()

	// Subscribe globally to trade.
	err := wsClient.Subscribe(ctx, []string{"trade"}, nil)
	require.NoError(t, err)

	waitForSID(t, wsClient, "trade", 10*time.Second)

	// Confirm data is flowing.
	collector.waitForType(t, "trade", 15*time.Second)

	// Unsubscribe.
	err = wsClient.Unsubscribe(ctx, []string{"trade"})
	require.NoError(t, err)

	// Clear and wait — should receive no more trade messages.
	collector.clear()
	time.Sleep(3 * time.Second)

	trades := collector.ofType("trade")
	assert.Empty(t, trades, "should not receive trades after unsubscribe")
}

// ---------------------------------------------------------------------------
// Subscription management operations
// ---------------------------------------------------------------------------

func TestWSIntegration_AddMarket(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()

	btc15m := getActiveBTC15MTicker(t, httpClient)
	btcd := getActiveBTCDTicker(t, httpClient)

	// Subscribe with BTC15M first.
	err := wsClient.AddMarkets(ctx, []string{btc15m}, []string{"orderbook_delta"})
	require.NoError(t, err)

	snapshot1 := collector.waitForType(t, "orderbook_snapshot", 15*time.Second)
	var body1 OrderbookSnapshotData
	require.NoError(t, json.Unmarshal(snapshot1.Msg, &body1))
	assert.Equal(t, btc15m, body1.MarketTicker)

	// Add BTCD market.
	err = wsClient.AddMarkets(ctx, []string{btcd}, []string{"orderbook_delta"})
	require.NoError(t, err)

	// Wait for BTCD snapshot.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snapshots := collector.ofType("orderbook_snapshot")
		for _, s := range snapshots {
			var body OrderbookSnapshotData
			if json.Unmarshal(s.Msg, &body) == nil && body.MarketTicker == btcd {
				t.Logf("received BTCD snapshot for %s", btcd)
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no orderbook_snapshot for %s within 15s", btcd)
}

func TestWSIntegration_RemoveMarket(t *testing.T) {
	httpClient, wsClient, collector, ctx, cancel := wsTestSetup(t)
	defer cancel()

	btc15m := getActiveBTC15MTicker(t, httpClient)
	btcd := getActiveBTCDTicker(t, httpClient)

	// Subscribe to both.
	err := wsClient.AddMarkets(ctx, []string{btc15m, btcd}, []string{"orderbook_delta"})
	require.NoError(t, err)
	time.Sleep(3 * time.Second) // let snapshots arrive

	// Remove BTCD.
	err = wsClient.RemoveMarkets(ctx, []string{btcd}, []string{"orderbook_delta"})
	require.NoError(t, err)
	collector.clear() // reset after removal

	// Collect for 3s — should only see BTC15M messages.
	time.Sleep(3 * time.Second)
	collector.mu.Lock()
	btcdCount := 0
	for _, m := range collector.raw {
		var body struct {
			Msg json.RawMessage `json:"msg"`
		}
		if json.Unmarshal(m, &body) == nil {
			var inner struct {
				MarketTicker string `json:"market_ticker"`
			}
			if json.Unmarshal(body.Msg, &inner) == nil && inner.MarketTicker == btcd {
				btcdCount++
			}
		}
	}
	collector.mu.Unlock()
	assert.Equal(t, 0, btcdCount, "should not receive messages for removed market %s", btcd)
}
