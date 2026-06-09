package gokalshi

import "encoding/json"

const wsPathSuffix = "/trade-api/ws/v2"

// ChannelState tracks the subscription state for a single WS channel.
type ChannelState struct {
	Name           string
	Markets        map[string]struct{}
	PendingMarkets map[string]struct{}
	SID            *int
	Seq            int
	Global         bool // true = subscribed without tickers (receives all markets)
}

// NewChannelState creates a new ChannelState for the given channel name.
func NewChannelState(name string) *ChannelState {
	return &ChannelState{
		Name:           name,
		Markets:        make(map[string]struct{}),
		PendingMarkets: make(map[string]struct{}),
	}
}

// WSMessage is the minimal envelope for incoming WebSocket messages.
type WSMessage struct {
	Type WSMessageType   `json:"type"`
	ID   int             `json:"id,omitempty"`
	SID  int             `json:"sid,omitempty"`
	Seq  int             `json:"seq,omitempty"`
	Msg  json.RawMessage `json:"msg,omitempty"`
}

// Outgoing command structs and data message types are in ws_messages_generated.go.
