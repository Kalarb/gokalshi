package gokalshi

// SubscribeOption configures optional parameters for Subscribe and AddMarkets.
type SubscribeOption func(*subscribeOpts)

type subscribeOpts struct {
	sendInitialSnapshot bool
}

func applyOpts(opts []SubscribeOption) subscribeOpts {
	var o subscribeOpts
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithInitialSnapshot requests an initial ticker snapshot from Kalshi on
// subscribe or add_markets. Without this, the ticker channel only sends
// updates on state changes (trades, BBO changes).
func WithInitialSnapshot() SubscribeOption {
	return func(o *subscribeOpts) { o.sendInitialSnapshot = true }
}
