package gokalshi

import "encoding/json"

const wsPathSuffix = "/trade-api/ws/v2"

// MsgTypeToChannel maps incoming WS message types to the channel(s) they can arrive on.
// Most types map to a single channel, but "event_lifecycle" can arrive on both
// "market_lifecycle_v2" and "multivariate_market_lifecycle".
// Also used as a set of known message types — actual routing uses SID.
var MsgTypeToChannel = map[WSMessageType][]string{
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
}

// ChannelState tracks the subscription state for a single WS channel.
type ChannelState struct {
	Name           string
	Markets        map[string]struct{}
	PendingMarkets map[string]struct{}
	SID            *int
	Seq            int
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

// Outgoing command structs are in ws_messages.go (WSCommand, SubscribeParams, etc.)
