package gokalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

// WSClient is the Kalshi WebSocket client. Each strategy gets its own instance.
type WSClient struct {
	cfg    *ClientConfig
	conn   *websocket.Conn
	logger *slog.Logger

	msgCh chan []byte // raw messages dispatched to FeedHandler

	mu          sync.Mutex
	msgID       int
	channels    map[string]*ChannelState
	sidMap      map[int]*ChannelState
	pendingInit map[int]string // msgID → channel name

	readLimit    int64 // max WS message size in bytes (0 = library default)
	minBackoff   time.Duration
	maxBackoff   time.Duration
	OnDisconnect func() // called when connection is lost (before reconnect)
	OnReconnect  func() // called after successful reconnect + subscription recovery

	closed         atomic.Bool
	forceReconnect atomic.Bool // set on sequence gap to trigger reconnect
}

// WSClientOption configures a WSClient.
type WSClientOption func(*WSClient)

// WithWSMsgBufSize sets the message channel buffer size.
func WithWSMsgBufSize(n int) WSClientOption {
	return func(c *WSClient) { c.msgCh = make(chan []byte, n) }
}

// WithWSReadLimit sets the maximum size of a single WebSocket message in bytes.
// The nhooyr.io/websocket library defaults to 32768 (32KB), which is too small
// for channels with many subscribed tickers. A value of 1<<20 (1MB) is
// recommended for production use.
func WithWSReadLimit(bytes int64) WSClientOption {
	return func(c *WSClient) { c.readLimit = bytes }
}

// WithWSBackoff sets min/max backoff durations for reconnection.
func WithWSBackoff(min, max time.Duration) WSClientOption {
	return func(c *WSClient) { c.minBackoff = min; c.maxBackoff = max }
}

// NewWSClient creates a new WebSocket client.
func NewWSClient(cfg *ClientConfig, opts ...WSClientOption) *WSClient {
	c := &WSClient{
		cfg:         cfg,
		msgCh:       make(chan []byte, 4096),
		logger:      newDiscardLogger(),
		channels:    make(map[string]*ChannelState),
		sidMap:      make(map[int]*ChannelState),
		pendingInit: make(map[int]string),
		minBackoff:  1 * time.Second,
		maxBackoff:  32 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// MsgCh returns the read-only channel for incoming messages.
func (c *WSClient) MsgCh() <-chan []byte { return c.msgCh }

// Connected reports whether the WebSocket connection is currently established.
func (c *WSClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Connect dials the Kalshi WebSocket endpoint with auth headers.
func (c *WSClient) Connect(ctx context.Context) error {
	url := c.cfg.WSBaseURL + wsPathSuffix

	headers, err := c.cfg.Credentials.RequestHeaders("GET", wsPathSuffix)
	if err != nil {
		return fmt.Errorf("generate WS auth headers: %w", err)
	}

	httpHeaders := http.Header{}
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: httpHeaders,
	})
	if err != nil {
		return fmt.Errorf("WS dial %s: %w", url, err)
	}

	if c.readLimit > 0 {
		conn.SetReadLimit(c.readLimit)
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	c.logger.LogAttrs(ctx, slog.LevelInfo, "ws_connected", slog.String("url", url))
	return nil
}

// Close gracefully closes the WebSocket connection.
// Safe for concurrent use.
func (c *WSClient) Close() error {
	c.closed.Store(true)
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		return conn.Close(websocket.StatusNormalClosure, "client closing")
	}
	return nil
}

// ListenLoop is the main read loop. It reads messages, handles reconnection
// with exponential backoff, and dispatches data messages on MsgCh.
func (c *WSClient) ListenLoop(ctx context.Context) {
	backoff := c.minBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		c.mu.Lock()
		noConn := c.conn == nil
		c.mu.Unlock()
		if noConn {
			if err := c.Connect(ctx); err != nil {
				c.logger.LogAttrs(ctx, slog.LevelError, "ws_connect_failed",
					slog.String("error", err.Error()), slog.String("retry_in", backoff.String()))
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				backoff = c.nextBackoff(backoff)
				continue
			}
		}

		err := c.readLoop(ctx)
		if ctx.Err() != nil || c.closed.Load() {
			return
		}

		c.logger.LogAttrs(ctx, slog.LevelError, "ws_disconnected", slog.String("error", err.Error()))
		c.handleConnectionLoss()

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		if err := c.Connect(ctx); err != nil {
			c.logger.LogAttrs(ctx, slog.LevelError, "ws_reconnect_failed", slog.String("error", err.Error()))
			backoff = c.nextBackoff(backoff)
			continue
		}

		backoff = c.minBackoff
		if err := c.recoverSubscriptions(ctx); err != nil {
			c.logger.LogAttrs(ctx, slog.LevelError, "ws_recovery_failed", slog.String("error", err.Error()))
		} else if c.OnReconnect != nil {
			c.OnReconnect()
		}
	}
}

func (c *WSClient) readLoop(ctx context.Context) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		c.handleIncoming(data)
		if c.forceReconnect.CompareAndSwap(true, false) {
			return fmt.Errorf("sequence gap detected, forcing reconnect")
		}
	}
}

func (c *WSClient) handleIncoming(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_invalid_json",
			slog.String("error", err.Error()))
		return
	}

	// Process under lock, but never hold mu during the blocking msgCh send.
	var dispatch []byte

	c.mu.Lock()
	switch msg.Type {
	case "subscribed":
		c.handleSubscribed(msg)
	case "ok":
		c.handleOK(msg)
	case "unsubscribed":
		c.handleUnsubscribed(msg)
	case "error":
		var errBody WSErrorBody
		if err := json.Unmarshal(msg.Msg, &errBody); err == nil {
			c.logger.LogAttrs(context.Background(), slog.LevelError, "ws_server_error",
				slog.Int("code", errBody.Code), slog.String("msg", errBody.Msg))
		} else {
			c.logger.LogAttrs(context.Background(), slog.LevelError, "ws_server_error",
				slog.String("raw", string(data)))
		}
	default:
		dispatch = c.handleDataMessage(msg, data)
	}
	c.mu.Unlock()

	// Dispatch outside the lock — blocking send provides backpressure without
	// holding mu, so AddMarkets/RemoveMarkets can still proceed.
	if dispatch != nil {
		c.msgCh <- dispatch
	}
}

func (c *WSClient) handleSubscribed(msg WSMessage) {
	channelName, ok := c.pendingInit[msg.ID]
	if !ok {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_unknown_msg_id",
			slog.Int("id", msg.ID))
		return
	}
	delete(c.pendingInit, msg.ID)

	// SID is nested inside msg body, not at the top level.
	var body SubscribedBody
	if err := json.Unmarshal(msg.Msg, &body); err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_parse_error",
			slog.String("event", "subscribed"), slog.String("error", err.Error()))
		return
	}

	state, ok := c.channels[channelName]
	if !ok {
		return
	}

	sid := body.SID
	state.SID = &sid
	c.sidMap[sid] = state

	// Flush pending markets. Loop handles tickers queued by concurrent
	// AddMarkets calls during the unlock window.
	for len(state.PendingMarkets) > 0 {
		tickers := make([]string, 0, len(state.PendingMarkets))
		for t := range state.PendingMarkets {
			tickers = append(tickers, t)
		}
		state.PendingMarkets = make(map[string]struct{})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// Must release lock before sending on WS.
		c.mu.Unlock()
		err := c.sendUpdateSub(ctx, sid, tickers, WSUpdateAddMarkets, false)
		cancel()
		c.mu.Lock()
		if err != nil {
			c.logger.LogAttrs(context.Background(), slog.LevelError, "ws_pending_flush_failed",
				slog.String("error", err.Error()))
			c.forceReconnect.Store(true)
			break
		}
	}

	c.logger.LogAttrs(context.Background(), slog.LevelInfo, "ws_subscribed",
		slog.String("channel", channelName), slog.Int("sid", sid))
}

func (c *WSClient) handleOK(msg WSMessage) {
	state, ok := c.sidMap[msg.SID]
	if !ok {
		return
	}

	if state.Seq != 0 && (state.Seq+1) != msg.Seq {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_sequence_gap",
			slog.String("channel", state.Name), slog.Int("expected", state.Seq+1), slog.Int("got", msg.Seq))
		c.forceReconnect.Store(true)
	}
	state.Seq = msg.Seq

	// Parse msg body for market reconciliation.
	if len(msg.Msg) > 0 {
		var body OkBody
		if err := json.Unmarshal(msg.Msg, &body); err == nil && len(body.MarketTickers) > 0 {
			state.Markets = make(map[string]struct{}, len(body.MarketTickers))
			for _, t := range body.MarketTickers {
				state.Markets[t] = struct{}{}
			}
		}
	}
}

func (c *WSClient) handleUnsubscribed(msg WSMessage) {
	delete(c.sidMap, msg.SID)
}

// handleDataMessage processes a data message under mu and returns the raw bytes
// to dispatch to msgCh (or nil if the message should be dropped).
func (c *WSClient) handleDataMessage(msg WSMessage, raw []byte) []byte {
	_, ok := MsgTypeToChannel[msg.Type]
	if !ok {
		c.logger.LogAttrs(context.Background(), slog.LevelDebug, "ws_unknown_message_type",
			slog.String("type", string(msg.Type)))
		return nil
	}

	state, ok := c.sidMap[msg.SID]
	if !ok {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_unknown_sid_dropped",
			slog.String("type", string(msg.Type)), slog.Int("sid", msg.SID))
		return nil
	}

	if state.Seq != 0 && (state.Seq+1) != msg.Seq {
		c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_sequence_gap",
			slog.String("channel", state.Name), slog.Int("expected", state.Seq+1), slog.Int("got", msg.Seq))
		c.forceReconnect.Store(true)
	}
	state.Seq = msg.Seq

	return raw
}

// Subscribe creates a new subscription for the given channels.
// Pass nil for tickers to subscribe globally (all markets).
// Pass a non-nil slice to subscribe to specific tickers only.
// Returns ErrInvalidArgument if a channel already has any subscription;
// call Unsubscribe first to replace an existing subscription.
func (c *WSClient) Subscribe(ctx context.Context, channels []string, tickers []string, opts ...SubscribeOption) error {
	if len(channels) == 0 {
		return fmt.Errorf("channels must not be empty: %w", ErrInvalidArgument)
	}
	if tickers != nil && len(tickers) == 0 {
		return fmt.Errorf("tickers must be nil (global) or non-empty: %w", ErrInvalidArgument)
	}
	o := applyOpts(opts)
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		if _, ok := c.channels[ch]; ok {
			return fmt.Errorf("channel %q already subscribed; call Unsubscribe first: %w", ch, ErrInvalidArgument)
		}

		state := NewChannelState(ch)
		state.Global = tickers == nil
		c.channels[ch] = state

		for _, t := range tickers {
			state.Markets[t] = struct{}{}
		}

		c.mu.Unlock()
		err := c.sendSubscribe(ctx, ch, tickers, o.sendInitialSnapshot)
		c.mu.Lock()
		if err != nil {
			delete(c.channels, ch)
			c.deletePendingInitForChannel(ch)
			return err
		}
	}
	return nil
}

// Unsubscribe tears down subscriptions for the given channels entirely.
// Works for both global and ticker-scoped subscriptions.
// channels must not be nil or empty.
func (c *WSClient) Unsubscribe(ctx context.Context, channels []string) error {
	if len(channels) == 0 {
		return fmt.Errorf("channels must not be empty: %w", ErrInvalidArgument)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		state, ok := c.channels[ch]
		if !ok {
			continue
		}

		if state.SID != nil {
			sid := *state.SID
			c.mu.Unlock()
			err := c.sendUnsubscribe(ctx, sid)
			c.mu.Lock()
			if err != nil {
				return err
			}
			delete(c.sidMap, sid)
		} else {
			// No SID yet — clean up any pending subscribe for this channel.
			c.deletePendingInitForChannel(ch)
		}
		delete(c.channels, ch)
	}
	return nil
}

// AddMarkets subscribes to the given channels for the given tickers.
// tickers and channels must not be nil or empty.
// If a channel has a global subscription, AddMarkets converts it to
// ticker-scoped (the server narrows the feed). A warning is logged.
func (c *WSClient) AddMarkets(ctx context.Context, tickers []string, channels []string, opts ...SubscribeOption) error {
	// Validate before acquiring mu -- no state access needed.
	if len(tickers) == 0 {
		return fmt.Errorf("tickers must not be empty: %w", ErrInvalidArgument)
	}
	if len(channels) == 0 {
		return fmt.Errorf("channels must not be empty: %w", ErrInvalidArgument)
	}
	o := applyOpts(opts)
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		state, ok := c.channels[ch]
		if ok && state.Global {
			c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_global_to_ticker_scoped",
				slog.String("channel", ch),
				slog.Any("tickers", tickers),
				slog.String("note", "AddMarkets on global subscription narrows it to specified tickers only"))
			state.Global = false
		}
		if !ok {
			state = NewChannelState(ch)
			c.channels[ch] = state
		}

		// Filter out already-subscribed tickers.
		var newTickers []string
		for _, t := range tickers {
			if _, exists := state.Markets[t]; !exists {
				newTickers = append(newTickers, t)
			}
		}
		if len(newTickers) == 0 {
			continue
		}

		if state.SID != nil {
			// Channel has a SID — send update_subscription.
			sid := *state.SID
			c.mu.Unlock()
			err := c.sendUpdateSub(ctx, sid, newTickers, WSUpdateAddMarkets, o.sendInitialSnapshot)
			c.mu.Lock()
			if err != nil {
				return err
			}
			for _, t := range newTickers {
				state.Markets[t] = struct{}{}
			}
		} else if _, pending := c.pendingInitForChannel(ch); pending {
			// Subscribe in progress -- queue tickers.
			for _, t := range newTickers {
				state.PendingMarkets[t] = struct{}{}
			}
		} else {
			// No SID, no pending -- send initial subscribe.
			c.mu.Unlock()
			err := c.sendSubscribe(ctx, ch, newTickers, o.sendInitialSnapshot)
			c.mu.Lock()
			if err != nil {
				if !ok {
					// Channel was newly created in this call — clean up.
					delete(c.channels, ch)
					c.deletePendingInitForChannel(ch)
				}
				return err
			}
			for _, t := range newTickers {
				state.Markets[t] = struct{}{}
			}
		}
	}
	return nil
}

// RemoveMarkets unsubscribes the given tickers from the given channels.
// tickers and channels must not be nil or empty.
// If a channel has a global subscription, RemoveMarkets converts it to
// ticker-scoped (the server narrows the feed). A warning is logged.
func (c *WSClient) RemoveMarkets(ctx context.Context, tickers []string, channels []string) error {
	// Validate before acquiring mu — no state access needed.
	if len(tickers) == 0 {
		return fmt.Errorf("tickers must not be empty: %w", ErrInvalidArgument)
	}
	if len(channels) == 0 {
		return fmt.Errorf("channels must not be empty: %w", ErrInvalidArgument)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		state, ok := c.channels[ch]
		if !ok || state.SID == nil {
			continue
		}
		if state.Global {
			c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_global_remove_markets",
				slog.String("channel", ch),
				slog.Any("tickers", tickers),
				slog.String("note", "RemoveMarkets on global subscription changes server-side behavior"))
			state.Global = false
		}

		sid := *state.SID
		c.mu.Unlock()
		err := c.sendUpdateSub(ctx, sid, tickers, WSUpdateDeleteMarkets, false)
		c.mu.Lock()
		if err != nil {
			return err
		}
		for _, t := range tickers {
			delete(state.Markets, t)
		}
	}
	return nil
}

func (c *WSClient) pendingInitForChannel(ch string) (int, bool) {
	for id, name := range c.pendingInit {
		if name == ch {
			return id, true
		}
	}
	return 0, false
}

func (c *WSClient) deletePendingInitForChannel(ch string) {
	for id, name := range c.pendingInit {
		if name == ch {
			delete(c.pendingInit, id)
			return
		}
	}
}

func (c *WSClient) handleConnectionLoss() {
	// Notify the strategy before clearing state — allows cancel-all + kill switch.
	if c.OnDisconnect != nil {
		c.OnDisconnect()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sidMap = make(map[int]*ChannelState)
	c.pendingInit = make(map[int]string)
	for _, state := range c.channels {
		state.SID = nil
		state.Seq = 0
	}
	c.conn = nil
}

func (c *WSClient) recoverSubscriptions(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, state := range c.channels {
		var tickers []string
		if state.Global {
			// Global subscription — recover with nil tickers.
			tickers = nil
		} else if len(state.Markets) > 0 {
			tickers = make([]string, 0, len(state.Markets))
			for t := range state.Markets {
				tickers = append(tickers, t)
			}
		} else {
			continue // empty ticker-scoped channel, nothing to recover
		}
		c.mu.Unlock()
		err := c.sendSubscribe(ctx, state.Name, tickers, false)
		c.mu.Lock()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *WSClient) sendSubscribe(ctx context.Context, channelName string, tickers []string, sendSnapshot bool) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	c.pendingInit[id] = channelName
	c.mu.Unlock()

	cmd := WSCommand{
		ID:  id,
		Cmd: WSCmdSubscribe,
		Params: SubscribeParams{
			Channels:            []string{channelName},
			MarketTickers:       tickers,
			SendInitialSnapshot: sendSnapshot,
		},
	}
	return c.writeJSON(ctx, cmd)
}

func (c *WSClient) sendUnsubscribe(ctx context.Context, sid int) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	c.mu.Unlock()

	cmd := WSCommand{
		ID:  id,
		Cmd: WSCmdUnsubscribe,
		Params: UnsubscribeParams{
			SIDs: []int{sid},
		},
	}
	return c.writeJSON(ctx, cmd)
}

func (c *WSClient) sendUpdateSub(ctx context.Context, sid int, tickers []string, action WSUpdateAction, sendSnapshot bool) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	c.mu.Unlock()

	cmd := WSCommand{
		ID:  id,
		Cmd: WSCmdUpdateSubscription,
		Params: UpdateSubParams{
			SIDs:                []int{sid},
			MarketTickers:       tickers,
			Action:              action,
			SendInitialSnapshot: sendSnapshot,
		},
	}
	return c.writeJSON(ctx, cmd)
}

func (c *WSClient) writeJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal WS command: %w", err)
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("ws not connected")
	}
	// Ensure writes don't block indefinitely on stalled connections.
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, data)
}

func (c *WSClient) nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > c.maxBackoff {
		return c.maxBackoff
	}
	return next
}
