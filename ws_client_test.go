package gokalshi

import (
	"context"
	"encoding/json"
	"fmt"
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

	msg := fmt.Sprintf(`{"type":"subscribed","id":1,"msg":{"channel":"orderbook_delta","sid":99}}`)
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

// ---------------------------------------------------------------------------
// Step 1: msgCh lifecycle + Close() wait
// ---------------------------------------------------------------------------

func TestWSClient_MsgCh_ClosedAfterListenLoopExits(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		// Keep connection alive until test is done.
		<-time.After(5 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg, WithWSBackoff(10*time.Millisecond, 50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	go ws.ListenLoop(ctx)
	time.Sleep(100 * time.Millisecond) // let it connect

	cancel()
	time.Sleep(200 * time.Millisecond) // let ListenLoop exit

	// MsgCh should be closed — receive returns ok=false.
	select {
	case _, ok := <-ws.MsgCh():
		assert.False(t, ok, "MsgCh should be closed after ListenLoop exits")
	case <-time.After(2 * time.Second):
		t.Fatal("MsgCh was not closed after ListenLoop exited")
	}
}

func TestWSClient_MsgCh_RangeExitsOnShutdown(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		<-time.After(5 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg, WithWSBackoff(10*time.Millisecond, 50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	go ws.ListenLoop(ctx)
	time.Sleep(100 * time.Millisecond)

	rangeExited := make(chan struct{})
	go func() {
		for range ws.MsgCh() {
			// drain
		}
		close(rangeExited)
	}()

	cancel()

	select {
	case <-rangeExited:
		// range loop exited because channel was closed
	case <-time.After(2 * time.Second):
		t.Fatal("range over MsgCh did not exit after ListenLoop shutdown")
	}
}

func TestWSClient_Close_WaitsForListenLoop(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		<-time.After(5 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg, WithWSBackoff(10*time.Millisecond, 50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenDone := make(chan struct{})
	go func() {
		ws.ListenLoop(ctx)
		close(listenDone)
	}()
	time.Sleep(100 * time.Millisecond) // let it connect

	// Close should wait for ListenLoop to finish.
	// The close handshake may error if the server-side handler already exited.
	_ = ws.Close()

	select {
	case <-listenDone:
		// ListenLoop has exited by the time Close returns
	case <-time.After(2 * time.Second):
		t.Fatal("ListenLoop still running after Close() returned")
	}
}

// ---------------------------------------------------------------------------
// Step 2: PendingMarkets recovery on reconnect
// ---------------------------------------------------------------------------

func TestWSClient_HandleConnectionLoss_MergesPendingMarkets(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("orderbook_delta")
	state.Markets["EXISTING-1"] = struct{}{}
	state.PendingMarkets["PENDING-1"] = struct{}{}
	state.PendingMarkets["PENDING-2"] = struct{}{}
	sid := 5
	state.SID = &sid
	state.Seq = 10
	ws.channels["orderbook_delta"] = state
	ws.sidMap[5] = state
	ws.pendingInit[99] = "orderbook_delta"

	ws.handleConnectionLoss()

	// PendingMarkets should have been merged into Markets before cleanup.
	assert.Contains(t, state.Markets, "EXISTING-1", "existing markets should be preserved")
	assert.Contains(t, state.Markets, "PENDING-1", "pending markets should be merged into Markets")
	assert.Contains(t, state.Markets, "PENDING-2", "pending markets should be merged into Markets")
	assert.Empty(t, state.PendingMarkets, "PendingMarkets should be cleared after merge")

	// Standard cleanup assertions
	assert.Nil(t, state.SID)
	assert.Equal(t, 0, state.Seq)
}

// ---------------------------------------------------------------------------
// Step 5: Read timeout on silent server
// ---------------------------------------------------------------------------

func TestWSClient_ReadLoop_TimeoutOnSilentServer(t *testing.T) {
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		// Accept connection but never send any data.
		<-time.After(30 * time.Second)
	})
	defer srv.Close()

	cfg := testWSConfig(t, srv.URL)
	ws := NewWSClient(cfg, WithWSReadTimeout(500*time.Millisecond))

	ctx := context.Background()
	err := ws.Connect(ctx)
	require.NoError(t, err)
	defer ws.Close()

	// readLoop should return an error within ~1s due to read timeout.
	done := make(chan error, 1)
	go func() {
		done <- ws.readLoop(ctx)
	}()

	select {
	case err := <-done:
		assert.Error(t, err, "readLoop should return error on silent server")
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not timeout on silent server")
	}
}

// ---------------------------------------------------------------------------
// Step 4: Unlock/relock SID validation
// ---------------------------------------------------------------------------

func TestWSClient_AddMarkets_SIDInvalidatedDuringUnlock(t *testing.T) {
	// Server that reads commands slowly, giving us time to simulate connection loss.
	cmdReceived := make(chan struct{})
	srv := mockWSServer(t, func(conn *websocket.Conn) {
		for {
			_, _, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			close(cmdReceived)
			// Block forever — simulates a slow/stuck write.
			<-time.After(10 * time.Second)
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
	state.Markets["OLD-TICK"] = struct{}{}
	ws.channels["ticker"] = state
	ws.sidMap[10] = state

	// AddMarkets will unlock mu to send, then re-lock. During the unlock window,
	// simulate connection loss which clears SID.
	go func() {
		<-cmdReceived
		ws.handleConnectionLoss()
	}()

	// AddMarkets sends update_subscription, during which handleConnectionLoss fires.
	_ = ws.AddMarkets(ctx, []string{"NEW-TICK"}, []string{"ticker"})

	ws.mu.Lock()
	hasSID := state.SID != nil
	_, hasNew := state.Markets["NEW-TICK"]
	ws.mu.Unlock()

	if !hasSID {
		// SID was cleared (connection loss happened). NEW-TICK should NOT be in Markets
		// because the subscription it was sent on no longer exists.
		assert.False(t, hasNew,
			"NEW-TICK should not be in Markets when SID was invalidated during send")
	}
	// If SID still present (race went the other way), the test is inconclusive — that's OK.
}

// ---------------------------------------------------------------------------
// Step 3: Sequence gap drops message
// ---------------------------------------------------------------------------

func TestWSClient_HandleDataMessage_SequenceGap_DropsMessage(t *testing.T) {
	cfg := testWSConfig(t, "http://localhost")
	ws := NewWSClient(cfg)

	state := NewChannelState("orderbook_delta")
	sid := 1
	state.SID = &sid
	state.Seq = 5
	ws.channels["orderbook_delta"] = state
	ws.sidMap[1] = state

	msg := WSMessage{
		Type: WSMsgOrderbookDelta,
		SID:  1,
		Seq:  10, // gap: expected 6
	}
	raw := []byte(`{"type":"orderbook_delta","sid":1,"seq":10,"msg":{}}`)

	ws.mu.Lock()
	result := ws.handleDataMessage(msg, raw)
	ws.mu.Unlock()

	assert.Nil(t, result, "out-of-sequence message should NOT be dispatched")
	assert.True(t, ws.forceReconnect.Load(), "forceReconnect should be set on sequence gap")
}
