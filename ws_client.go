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

	c.conn = conn
	c.logger.LogAttrs(ctx, slog.LevelInfo, "ws_connected", slog.String("url", url))
	return nil
}

// Close gracefully closes the WebSocket connection.
// Safe for concurrent use.
func (c *WSClient) Close() error {
	c.closed.Store(true)
	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "client closing")
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

		if c.conn == nil {
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
	for {
		_, data, err := c.conn.Read(ctx)
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

	c.mu.Lock()
	defer c.mu.Unlock()

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
		c.handleDataMessage(msg, data)
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

	// Flush pending markets.
	if len(state.PendingMarkets) > 0 {
		tickers := make([]string, 0, len(state.PendingMarkets))
		for t := range state.PendingMarkets {
			tickers = append(tickers, t)
		}
		state.PendingMarkets = make(map[string]struct{})
		// Must release lock before sending on WS.
		c.mu.Unlock()
		_ = c.sendUpdateSub(context.Background(), sid, tickers, WSUpdateAddMarkets)
		c.mu.Lock()
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

func (c *WSClient) handleDataMessage(msg WSMessage, raw []byte) {
	_, ok := MsgTypeToChannel[msg.Type]
	if !ok {
		c.logger.LogAttrs(context.Background(), slog.LevelDebug, "ws_unknown_message_type",
			slog.String("type", string(msg.Type)))
		return
	}

	state, ok := c.sidMap[msg.SID]
	if ok {
		if state.Seq != 0 && (state.Seq+1) != msg.Seq {
			c.logger.LogAttrs(context.Background(), slog.LevelWarn, "ws_sequence_gap",
				slog.String("channel", state.Name), slog.Int("expected", state.Seq+1), slog.Int("got", msg.Seq))
			c.forceReconnect.Store(true)
		}
		state.Seq = msg.Seq
	}

	// Dispatch to FeedHandler (blocking — backpressure is preferable to
	// silently dropping messages and trading on corrupt data).
	c.msgCh <- raw
}

// AddMarkets subscribes to the given channels for the given tickers.
func (c *WSClient) AddMarkets(ctx context.Context, tickers []string, channels []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		state, ok := c.channels[ch]
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
			for _, t := range newTickers {
				state.Markets[t] = struct{}{}
			}
			c.mu.Unlock()
			err := c.sendUpdateSub(ctx, *state.SID, newTickers, WSUpdateAddMarkets)
			c.mu.Lock()
			if err != nil {
				return err
			}
		} else if _, pending := c.pendingInitForChannel(ch); pending {
			// Subscribe in progress — queue tickers.
			for _, t := range newTickers {
				state.PendingMarkets[t] = struct{}{}
			}
		} else {
			// No SID, no pending — send initial subscribe.
			for _, t := range newTickers {
				state.Markets[t] = struct{}{}
			}
			c.mu.Unlock()
			err := c.sendSubscribe(ctx, ch, newTickers)
			c.mu.Lock()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RemoveMarkets unsubscribes the given tickers from the given channels.
func (c *WSClient) RemoveMarkets(ctx context.Context, tickers []string, channels []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range channels {
		state, ok := c.channels[ch]
		if !ok || state.SID == nil {
			continue
		}

		for _, t := range tickers {
			delete(state.Markets, t)
		}

		c.mu.Unlock()
		err := c.sendUpdateSub(ctx, *state.SID, tickers, WSUpdateDeleteMarkets)
		c.mu.Lock()
		if err != nil {
			return err
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
		if len(state.Markets) == 0 {
			continue
		}
		tickers := make([]string, 0, len(state.Markets))
		for t := range state.Markets {
			tickers = append(tickers, t)
		}
		c.mu.Unlock()
		err := c.sendSubscribe(ctx, state.Name, tickers)
		c.mu.Lock()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *WSClient) sendSubscribe(ctx context.Context, channelName string, tickers []string) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	c.pendingInit[id] = channelName
	c.mu.Unlock()

	cmd := WSCommand{
		ID:  id,
		Cmd: WSCmdSubscribe,
		Params: SubscribeParams{
			Channels:      []string{channelName},
			MarketTickers: tickers,
		},
	}
	return c.writeJSON(ctx, cmd)
}

func (c *WSClient) sendUpdateSub(ctx context.Context, sid int, tickers []string, action WSUpdateAction) error {
	c.mu.Lock()
	c.msgID++
	id := c.msgID
	c.mu.Unlock()

	cmd := WSCommand{
		ID:  id,
		Cmd: WSCmdUpdateSubscription,
		Params: UpdateSubParams{
			SIDs:          []int{sid},
			MarketTickers: tickers,
			Action:        action,
		},
	}
	return c.writeJSON(ctx, cmd)
}

func (c *WSClient) writeJSON(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal WS command: %w", err)
	}
	if c.conn == nil {
		return fmt.Errorf("ws not connected")
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *WSClient) nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > c.maxBackoff {
		return c.maxBackoff
	}
	return next
}
