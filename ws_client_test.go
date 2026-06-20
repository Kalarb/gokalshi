package gokalshi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func testWSConfig(t *testing.T, serverURL string) *ClientConfig {
	t.Helper()
	cfg := testClientConfig(t, serverURL)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	cfg.WSBaseURL = wsURL
	return cfg
}

// mockWSServer creates an httptest server that upgrades to WebSocket.
func mockWSServer(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Logf("ws accept error: %v", err)
			return
		}
		defer conn.CloseNow()
		handler(conn)
	}))
}

func TestNewWSClient(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	assert.NotNil(t, ws.msgCh)
	assert.Equal(t, 4096, cap(ws.msgCh))
	assert.NotNil(t, ws.channels)
	assert.NotNil(t, ws.sidMap)
	assert.NotNil(t, ws.pendingInit)
	assert.Equal(t, 1*time.Second, ws.minBackoff)
	assert.Equal(t, 32*time.Second, ws.maxBackoff)
}

func TestWSClient_Options(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg,
		WithWSMsgBufSize(64),
		WithWSBackoff(100*time.Millisecond, 5*time.Second),
	)

	assert.Equal(t, 64, cap(ws.msgCh))
	assert.Equal(t, 100*time.Millisecond, ws.minBackoff)
	assert.Equal(t, 5*time.Second, ws.maxBackoff)
}

func TestWSClient_MsgCh(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	ch := ws.MsgCh()
	assert.NotNil(t, ch)
}

func TestWSClient_Connect(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		<-time.After(2 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	assert.NotNil(t, ws.conn)

	ws.Close()
}

func TestWSClient_Connected(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		<-time.After(2 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)

	assert.False(t, ws.Connected(), "should be false before Connect")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, ws.Connect(ctx))
	assert.True(t, ws.Connected(), "should be true after Connect")

	ws.Close()
	assert.False(t, ws.Connected(), "should be false after Close")
}

func TestWSClient_Close(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.Close()
	assert.NoError(t, err)
	assert.True(t, ws.closed.Load())
}

func TestWSClient_Close_Concurrent(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = ws.Close()
		}()
		go func() {
			defer wg.Done()
			_ = ws.closed.Load()
		}()
	}
	wg.Wait()
}

func TestWSClient_HandleIncoming_DataMessage(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("orderbook_delta")
	sid := 1
	state.SID = &sid
	ws.channels["orderbook_delta"] = state
	ws.sidMap[1] = state

	msg := `{"type":"orderbook_delta","sid":1,"seq":1,"msg":{"market_ticker":"TEST"}}`
	ws.handleIncoming([]byte(msg))

	select {
	case raw := <-ws.MsgCh():
		assert.Contains(t, string(raw), "orderbook_delta")
	case <-time.After(time.Second):
		t.Fatal("message not dispatched")
	}
}

func TestWSClient_HandleIncoming_DataMessage_UnknownSID(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// No SID 99 in sidMap — simulates a message arriving after unsubscribe.
	msg := `{"type":"trade","sid":99,"seq":1,"msg":{"market_ticker":"TEST"}}`
	ws.handleIncoming([]byte(msg))

	select {
	case <-ws.MsgCh():
		t.Fatal("message with unknown SID should have been dropped")
	case <-time.After(200 * time.Millisecond):
		// Expected: message was dropped.
	}
}

func TestWSClient_HandleIncoming_Subscribed(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	ws.channels["ticker"] = NewChannelState("ticker")
	ws.pendingInit[42] = "ticker"

	msg := `{"type":"subscribed","id":42,"msg":{"channel":"ticker","sid":5}}`
	ws.handleIncoming([]byte(msg))

	assert.NotNil(t, ws.channels["ticker"].SID)
	assert.Equal(t, 5, *ws.channels["ticker"].SID)
	assert.NotNil(t, ws.sidMap[5])
	assert.Empty(t, ws.pendingInit)
}

func TestWSClient_HandleIncoming_OK(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("trade")
	sid := 3
	state.SID = &sid
	ws.sidMap[3] = state

	msg := `{"type":"ok","sid":3,"seq":5}`
	ws.handleIncoming([]byte(msg))

	assert.Equal(t, 5, state.Seq)
}

func TestWSClient_HandleIncoming_SequenceGap(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("trade")
	sid := 3
	state.SID = &sid
	state.Seq = 5
	ws.sidMap[3] = state

	msg := `{"type":"ok","sid":3,"seq":10}`
	ws.handleIncoming([]byte(msg))

	assert.Equal(t, 10, state.Seq)
}

func TestWSClient_HandleIncoming_Unsubscribed(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("fill")
	sid := 7
	state.SID = &sid
	ws.sidMap[7] = state

	msg := `{"type":"unsubscribed","sid":7}`
	ws.handleIncoming([]byte(msg))

	_, exists := ws.sidMap[7]
	assert.False(t, exists)
}

func TestWSClient_HandleIncoming_InvalidJSON(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	ws.handleIncoming([]byte("not json"))
}

func TestWSClient_HandleIncoming_UnknownType(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	msg := `{"type":"totally_unknown","sid":1,"seq":1}`
	ws.handleIncoming([]byte(msg))

	select {
	case <-ws.MsgCh():
		t.Fatal("should not dispatch unknown types")
	default:
	}
}

func TestWSClient_HandleIncoming_Error(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	msg := `{"type":"error","msg":"bad request"}`
	ws.handleIncoming([]byte(msg))
}

func TestWSClient_AddMarkets_NewChannel(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	err = ws.AddMarkets(ctx, []string{"TICK-1"}, []string{"orderbook_delta"})
	require.NoError(t, err)

	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "subscribe", cmd["cmd"])
	case <-time.After(2 * time.Second):
		t.Fatal("no subscribe command sent")
	}
}

func TestWSClient_AddMarkets_ExistingSID(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	state := NewChannelState("ticker")
	sid := 10
	state.SID = &sid
	state.Markets["TICK-OLD"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[10] = state

	err = ws.AddMarkets(ctx, []string{"TICK-NEW"}, []string{"ticker"})
	require.NoError(t, err)

	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
	case <-time.After(2 * time.Second):
		t.Fatal("no update_subscription command sent")
	}
}

func TestWSClient_RemoveMarkets(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	state := NewChannelState("fill")
	sid := 20
	state.SID = &sid
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["fill"] = state
	ws.sidMap[20] = state

	err = ws.RemoveMarkets(ctx, []string{"TICK-1"}, []string{"fill"})
	require.NoError(t, err)

	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		assert.Equal(t, "delete_markets", params["action"])
	case <-time.After(2 * time.Second):
		t.Fatal("no delete_markets command sent")
	}
}

func TestWSClient_GetSnapshot(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	state := NewChannelState("ticker")
	sid := 30
	state.SID = &sid
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[30] = state

	err = ws.GetSnapshot(ctx, []string{"TICK-1"}, []string{"ticker"})
	require.NoError(t, err)

	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		assert.Equal(t, "get_snapshot", params["action"])
		assert.Equal(t, []any{"TICK-1"}, params["market_tickers"])
		assert.Equal(t, []any{float64(30)}, params["sids"])
	case <-time.After(2 * time.Second):
		t.Fatal("no get_snapshot command sent")
	}
}

func TestWSClient_GetSnapshot_NoSIDSkips(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)
	ctx := context.Background()

	// Channel exists but has no SID (not yet subscribed) -> skipped, no error.
	ws.channels["ticker"] = NewChannelState("ticker")

	err := ws.GetSnapshot(ctx, []string{"TICK-1"}, []string{"ticker"})
	require.NoError(t, err)
}

func TestWSClient_GetSnapshot_Validation(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.ErrorIs(t, ws.GetSnapshot(ctx, nil, []string{"ticker"}), ErrInvalidArgument)
	require.ErrorIs(t, ws.GetSnapshot(ctx, []string{"TICK-1"}, nil), ErrInvalidArgument)
}

func TestWSClient_HandleConnectionLoss(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("ticker")
	sid := 5
	state.SID = &sid
	state.Seq = 42
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[5] = state
	ws.pendingInit[99] = "ticker"

	ws.handleConnectionLoss()

	assert.Empty(t, ws.sidMap)
	assert.Empty(t, ws.pendingInit)
	assert.Nil(t, state.SID)
	assert.Equal(t, 0, state.Seq)
	assert.Contains(t, state.Markets, "TICK-1")
	assert.Nil(t, ws.conn)
}

func TestWSClient_NextBackoff(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg, WithWSBackoff(1*time.Second, 32*time.Second))

	assert.Equal(t, 2*time.Second, ws.nextBackoff(1*time.Second))
	assert.Equal(t, 4*time.Second, ws.nextBackoff(2*time.Second))
	assert.Equal(t, 8*time.Second, ws.nextBackoff(4*time.Second))
	assert.Equal(t, 16*time.Second, ws.nextBackoff(8*time.Second))
	assert.Equal(t, 32*time.Second, ws.nextBackoff(16*time.Second))
	assert.Equal(t, 32*time.Second, ws.nextBackoff(32*time.Second))
}

func TestNewChannelState(t *testing.T) {
	state := NewChannelState("orderbook_delta")
	assert.Equal(t, "orderbook_delta", state.Name)
	assert.NotNil(t, state.Markets)
	assert.NotNil(t, state.PendingMarkets)
	assert.Nil(t, state.SID)
	assert.Equal(t, 0, state.Seq)
}

func TestWSClient_WriteJSON_NotConnected(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.writeJSON(context.Background(), map[string]any{"test": true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestWSClient_PendingInitForChannel(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	ws.pendingInit[1] = "ticker"
	ws.pendingInit[2] = "trade"

	id, found := ws.pendingInitForChannel("ticker")
	assert.True(t, found)
	assert.Equal(t, 1, id)

	_, found = ws.pendingInitForChannel("nonexistent")
	assert.False(t, found)
}

func TestWSClient_AddMarkets_AlreadySubscribed(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		<-time.After(2 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	state := NewChannelState("ticker")
	sid := 1
	state.SID = &sid
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[1] = state

	err = ws.AddMarkets(ctx, []string{"TICK-1"}, []string{"ticker"})
	require.NoError(t, err)
}

func TestWSClient_ListenLoop_ContextCancel(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		conn.Write(context.Background(), websocket.MessageText,
			[]byte(`{"type":"orderbook_snapshot","sid":1,"seq":1}`))
		<-time.After(5 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg, WithWSBackoff(10*time.Millisecond, 50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		ws.ListenLoop(ctx)
		close(done)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ListenLoop did not exit after context cancel")
	}
}

func TestMsgTypeToChannel(t *testing.T) {
	expected := map[WSMessageType][]string{
		WSMsgOrderbookSnapshot:           {"orderbook_delta"},
		WSMsgOrderbookDelta:              {"orderbook_delta"},
		WSMsgTicker:                      {"ticker"},
		WSMsgTrade:                       {"trade"},
		WSMsgFill:                        {"fill"},
		WSMsgMarketPosition:              {"market_positions"},
		WSMsgMarketLifecycleV2:           {"market_lifecycle_v2"},
		WSMsgEventLifecycle:              {"market_lifecycle_v2", "multivariate_market_lifecycle"},
		WSMsgMultivariateMarketLifecycle: {"multivariate_market_lifecycle"},
		WSMsgMultivariateLookup:          {"multivariate"},
		WSMsgUserOrder:                   {"user_orders"},
		WSMsgOrderGroupUpdates:           {"order_group_updates"},
		WSMsgRFQCreated:                  {"communications"},
		WSMsgRFQDeleted:                  {"communications"},
		WSMsgQuoteCreated:                {"communications"},
		WSMsgQuoteAccepted:               {"communications"},
		WSMsgQuoteExecuted:               {"communications"},
		WSMsgEventFeeUpdate:              {"market_lifecycle_v2"},
		WSMsgCfbenchmarksValue:           {"cfbenchmarks_value"},
		WSMsgCfbenchmarksValueIndexlist:  {"cfbenchmarks_value"},
	}

	for msgType, expectedChannels := range expected {
		channels, ok := MsgTypeToChannel[msgType]
		assert.True(t, ok, "missing mapping for %s", msgType)
		assert.Equal(t, expectedChannels, channels, "wrong channels for %s", msgType)
	}

	assert.Len(t, MsgTypeToChannel, len(expected))
}

func TestWSClient_HandleIncoming_SubscribedFlushPending(t *testing.T) {
	sent := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			sent <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	state := NewChannelState("orderbook_delta")
	state.PendingMarkets["PEND-1"] = struct{}{}
	state.PendingMarkets["PEND-2"] = struct{}{}
	ws.channels["orderbook_delta"] = state
	ws.pendingInit[1] = "orderbook_delta"

	msg := `{"type":"subscribed","id":1,"msg":{"channel":"orderbook_delta","sid":99}}`
	ws.handleIncoming([]byte(msg))

	select {
	case raw := <-sent:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
	case <-time.After(2 * time.Second):
		t.Fatal("pending markets not flushed")
	}

	assert.Empty(t, state.PendingMarkets)
}

func TestWSClient_AddRemoveMarkets_InvalidArgs(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)
	ctx := context.Background()

	cases := []struct {
		name     string
		method   string
		tickers  []string
		channels []string
		wantMsg  string
	}{
		{"AddMarkets/nil_tickers", "add", nil, []string{"ticker"}, "tickers must not be empty"},
		{"AddMarkets/empty_tickers", "add", []string{}, []string{"ticker"}, "tickers must not be empty"},
		{"AddMarkets/nil_channels", "add", []string{"TICK-1"}, nil, "channels must not be empty"},
		{"AddMarkets/empty_channels", "add", []string{"TICK-1"}, []string{}, "channels must not be empty"},
		{"RemoveMarkets/nil_tickers", "remove", nil, []string{"ticker"}, "tickers must not be empty"},
		{"RemoveMarkets/empty_tickers", "remove", []string{}, []string{"ticker"}, "tickers must not be empty"},
		{"RemoveMarkets/nil_channels", "remove", []string{"TICK-1"}, nil, "channels must not be empty"},
		{"RemoveMarkets/empty_channels", "remove", []string{"TICK-1"}, []string{}, "channels must not be empty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.method == "add" {
				err = ws.AddMarkets(ctx, tc.tickers, tc.channels)
			} else {
				err = ws.RemoveMarkets(ctx, tc.tickers, tc.channels)
			}
			require.ErrorIs(t, err, ErrInvalidArgument)
			assert.ErrorContains(t, err, tc.wantMsg)
		})
	}
}

// ---------------------------------------------------------------------------
// Subscribe / Unsubscribe
// ---------------------------------------------------------------------------

func TestWSClient_Subscribe_Global(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	err := ws.Subscribe(ctx, []string{"market_lifecycle_v2"}, nil)
	require.NoError(t, err)

	// Verify the subscribe command has no market_tickers field.
	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "subscribe", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		assert.Equal(t, []any{"market_lifecycle_v2"}, params["channels"])
		_, hasTickers := params["market_tickers"]
		assert.False(t, hasTickers, "global subscribe must not include market_tickers")
	case <-time.After(2 * time.Second):
		t.Fatal("no subscribe command sent")
	}

	// Verify channel state.
	state, ok := ws.channels["market_lifecycle_v2"]
	require.True(t, ok)
	assert.True(t, state.Global)
	assert.Empty(t, state.Markets)
}

func TestWSClient_Subscribe_TickerScoped(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	err := ws.Subscribe(ctx, []string{"trade"}, []string{"TICK-1", "TICK-2"})
	require.NoError(t, err)

	// Verify the subscribe command includes market_tickers.
	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "subscribe", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		assert.Equal(t, []any{"trade"}, params["channels"])
		tickers := params["market_tickers"].([]any)
		assert.ElementsMatch(t, []any{"TICK-1", "TICK-2"}, tickers)
	case <-time.After(2 * time.Second):
		t.Fatal("no subscribe command sent")
	}

	// Verify channel state.
	state, ok := ws.channels["trade"]
	require.True(t, ok)
	assert.False(t, state.Global)
	assert.Contains(t, state.Markets, "TICK-1")
	assert.Contains(t, state.Markets, "TICK-2")
}

func TestWSClient_Subscribe_AlreadySubscribed(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// Set up an existing global channel.
	state := NewChannelState("trade")
	state.Global = true
	ws.channels["trade"] = state

	// Subscribe again — should error.
	err := ws.Subscribe(context.Background(), []string{"trade"}, nil)
	require.ErrorIs(t, err, ErrInvalidArgument)
	assert.ErrorContains(t, err, "already subscribed")
}

func TestWSClient_Subscribe_AlreadySubscribed_TickerScoped(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// Set up an existing ticker-scoped channel.
	state := NewChannelState("ticker")
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = state

	// Subscribe again — should error regardless of tickers.
	err := ws.Subscribe(context.Background(), []string{"ticker"}, []string{"TICK-1"})
	require.ErrorIs(t, err, ErrInvalidArgument)
	assert.ErrorContains(t, err, "already subscribed")
}

func TestWSClient_Subscribe_InvalidArgs(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)
	ctx := context.Background()

	cases := []struct {
		name     string
		channels []string
	}{
		{"nil_channels", nil},
		{"empty_channels", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ws.Subscribe(ctx, tc.channels, nil)
			require.ErrorIs(t, err, ErrInvalidArgument)
			assert.ErrorContains(t, err, "channels must not be empty")
		})
	}
}

func TestWSClient_Subscribe_EmptyTickers(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.Subscribe(context.Background(), []string{"trade"}, []string{})
	require.ErrorIs(t, err, ErrInvalidArgument)
	assert.ErrorContains(t, err, "tickers must be nil (global) or non-empty")
}

func TestWSClient_Subscribe_ErrorCleansUpState(t *testing.T) {
	// Don't connect — writeJSON will fail, causing sendSubscribe to error.
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.Subscribe(context.Background(), []string{"trade"}, nil)
	require.Error(t, err)

	// Verify the channel state was cleaned up.
	assert.Empty(t, ws.channels, "channels should be empty after failed Subscribe")
	assert.Empty(t, ws.pendingInit, "pendingInit should be empty after failed Subscribe")
}

func TestWSClient_AddMarkets_ErrorCleansUpNewChannel(t *testing.T) {
	// Don't connect — writeJSON will fail for the new channel's sendSubscribe.
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.AddMarkets(context.Background(), []string{"TICK-1"}, []string{"trade"})
	require.Error(t, err)

	// Verify the newly created channel state was cleaned up.
	_, exists := ws.channels["trade"]
	assert.False(t, exists, "new channel should be cleaned up after failed AddMarkets")
	assert.Empty(t, ws.pendingInit, "pendingInit should be empty after failed AddMarkets")
}

func TestWSClient_Unsubscribe_PendingCleanup(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// Simulate a pending subscribe: channel exists, no SID, pendingInit has an entry.
	state := NewChannelState("trade")
	state.Global = true
	ws.channels["trade"] = state
	ws.pendingInit[42] = "trade"

	err := ws.Unsubscribe(context.Background(), []string{"trade"})
	require.NoError(t, err)

	_, chExists := ws.channels["trade"]
	assert.False(t, chExists, "channel should be removed")
	_, piExists := ws.pendingInit[42]
	assert.False(t, piExists, "pendingInit entry should be cleaned up")
}

func TestWSClient_AddMarkets_GlobalToTickerScoped(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	// Set up a global channel with SID.
	state := NewChannelState("trade")
	state.Global = true
	sid := 1
	state.SID = &sid
	ws.channels["trade"] = state
	ws.sidMap[1] = state

	// AddMarkets on global channel — should succeed with warning.
	err := ws.AddMarkets(ctx, []string{"TICK-1"}, []string{"trade"})
	require.NoError(t, err)

	// Verify update_subscription sent.
	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
	case <-time.After(2 * time.Second):
		t.Fatal("no update_subscription command sent")
	}

	// Verify state converted to ticker-scoped.
	assert.False(t, state.Global)
	assert.Contains(t, state.Markets, "TICK-1")
}

func TestWSClient_RemoveMarkets_GlobalWarning(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	// Set up a global channel with SID.
	state := NewChannelState("trade")
	state.Global = true
	sid := 1
	state.SID = &sid
	ws.channels["trade"] = state
	ws.sidMap[1] = state

	// RemoveMarkets on global channel — should succeed with warning.
	err := ws.RemoveMarkets(ctx, []string{"TICK-1"}, []string{"trade"})
	require.NoError(t, err)

	// Verify delete_markets sent.
	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "update_subscription", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		assert.Equal(t, "delete_markets", params["action"])
	case <-time.After(2 * time.Second):
		t.Fatal("no delete_markets command sent")
	}

	// Verify state converted from global.
	assert.False(t, state.Global)
}

func TestWSClient_Unsubscribe(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	// Set up a global channel with SID.
	state := NewChannelState("trade")
	state.Global = true
	sid := 5
	state.SID = &sid
	ws.channels["trade"] = state
	ws.sidMap[5] = state

	err := ws.Unsubscribe(ctx, []string{"trade"})
	require.NoError(t, err)

	// Verify unsubscribe command sent.
	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "unsubscribe", cmd["cmd"])
		params := cmd["params"].(map[string]any)
		sids := params["sids"].([]any)
		assert.Equal(t, float64(5), sids[0])
	case <-time.After(2 * time.Second):
		t.Fatal("no unsubscribe command sent")
	}

	// Verify state cleaned up.
	_, chExists := ws.channels["trade"]
	assert.False(t, chExists)
	_, sidExists := ws.sidMap[5]
	assert.False(t, sidExists)
}

func TestWSClient_Unsubscribe_TickerScoped(t *testing.T) {
	done := make(chan []byte, 5)
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			done <- data
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	// Set up a ticker-scoped channel.
	state := NewChannelState("ticker")
	sid := 10
	state.SID = &sid
	state.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[10] = state

	err := ws.Unsubscribe(ctx, []string{"ticker"})
	require.NoError(t, err)

	select {
	case raw := <-done:
		var cmd map[string]any
		require.NoError(t, json.Unmarshal(raw, &cmd))
		assert.Equal(t, "unsubscribe", cmd["cmd"])
	case <-time.After(2 * time.Second):
		t.Fatal("no unsubscribe command sent")
	}

	_, chExists := ws.channels["ticker"]
	assert.False(t, chExists)
}

func TestWSClient_Unsubscribe_NotSubscribed(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// Unsubscribing from a channel that doesn't exist — no error, just a no-op.
	err := ws.Unsubscribe(context.Background(), []string{"nonexistent"})
	require.NoError(t, err)
}

func TestWSClient_Unsubscribe_NoSID(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	// Channel exists but has no SID yet (subscribe pending).
	state := NewChannelState("trade")
	state.Global = true
	ws.channels["trade"] = state

	err := ws.Unsubscribe(context.Background(), []string{"trade"})
	require.NoError(t, err)

	_, exists := ws.channels["trade"]
	assert.False(t, exists, "channel state should be removed")
}

func TestWSClient_Unsubscribe_InvalidArgs(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	err := ws.Unsubscribe(context.Background(), nil)
	require.ErrorIs(t, err, ErrInvalidArgument)

	err = ws.Unsubscribe(context.Background(), []string{})
	require.ErrorIs(t, err, ErrInvalidArgument)
}

func TestWSClient_RecoverSubscriptions_Global(t *testing.T) {
	var mu sync.Mutex
	var cmds []map[string]any
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			var cmd map[string]any
			if json.Unmarshal(data, &cmd) == nil {
				mu.Lock()
				cmds = append(cmds, cmd)
				mu.Unlock()
			}
		}
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg)
	ctx := context.Background()

	require.NoError(t, ws.Connect(ctx))
	defer ws.Close()

	// Set up a global channel (SID cleared as if after disconnect).
	globalState := NewChannelState("market_lifecycle_v2")
	globalState.Global = true
	ws.channels["market_lifecycle_v2"] = globalState

	// Set up a ticker-scoped channel.
	tickerState := NewChannelState("ticker")
	tickerState.Markets["TICK-1"] = struct{}{}
	ws.channels["ticker"] = tickerState

	err := ws.recoverSubscriptions(ctx)
	require.NoError(t, err)

	// Wait for commands to arrive.
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, cmds, 2, "expected 2 subscribe commands (global + ticker-scoped)")

	// Find the global subscribe and verify no market_tickers.
	var foundGlobal, foundTicker bool
	for _, cmd := range cmds {
		params := cmd["params"].(map[string]any)
		channels := params["channels"].([]any)
		ch := channels[0].(string)
		if ch == "market_lifecycle_v2" {
			foundGlobal = true
			_, hasTickers := params["market_tickers"]
			assert.False(t, hasTickers, "global recovery must not include market_tickers")
		}
		if ch == "ticker" {
			foundTicker = true
			tickers := params["market_tickers"].([]any)
			assert.Contains(t, tickers, "TICK-1")
		}
	}
	assert.True(t, foundGlobal, "global channel not recovered")
	assert.True(t, foundTicker, "ticker-scoped channel not recovered")
}
