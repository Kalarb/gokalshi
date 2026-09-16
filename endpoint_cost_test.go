package gokalshi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// liveEndpointCostsPayload is the response GET /account/endpoint_costs actually
// returned from DEMO on 2026-09-15, reproduced verbatim.
//
// The previous tests built cost patterns by hand with the placeholders already
// substituted, so they exercised the matcher and never the construction — where
// the bug was. Nine of these seventeen entries use ":param", which the old
// {param}-only conversion could not match, so they silently billed at the
// default cost. Driving the real payload through ConfigureRateLimits is the
// only shape of test that would have caught it.
const liveEndpointCostsPayload = `{
  "default_cost": 10,
  "endpoint_costs": [
    {"method":"POST","path":"/trade-api/v2/account/api_usage_level/upgrade","cost":30},
    {"method":"GET","path":"/trade-api/v2/cfbenchmarks","cost":50},
    {"method":"GET","path":"/trade-api/v2/cfbenchmarks/*endpoint","cost":50},
    {"method":"POST","path":"/trade-api/v2/communications/quotes","cost":2},
    {"method":"DELETE","path":"/trade-api/v2/communications/quotes/:quote_id","cost":2},
    {"method":"GET","path":"/trade-api/v2/communications/quotes/:quote_id","cost":2},
    {"method":"DELETE","path":"/trade-api/v2/communications/rfqs/:rfq_id/quotes/:quote_id","cost":2},
    {"method":"GET","path":"/trade-api/v2/communications/rfqs/:rfq_id/quotes/:quote_id","cost":2},
    {"method":"PUT","path":"/trade-api/v2/communications/rfqs/:rfq_id/quotes/:quote_id/confirm","cost":1},
    {"method":"GET","path":"/trade-api/v2/margin/balance","cost":5},
    {"method":"DELETE","path":"/trade-api/v2/margin/orders/:order_id","cost":1},
    {"method":"POST","path":"/trade-api/v2/margin/orders/:order_id/decrease","cost":1},
    {"method":"DELETE","path":"/trade-api/v2/portfolio/events/orders","cost":2},
    {"method":"DELETE","path":"/trade-api/v2/portfolio/events/orders/:order_id","cost":2},
    {"method":"DELETE","path":"/trade-api/v2/portfolio/events/orders/batched","cost":2},
    {"method":"GET","path":"/trade-api/v2/portfolio/orders/:order_id","cost":2},
    {"method":"POST","path":"/trade-api/v2/portfolio/subaccounts/positions/transfer/batched","cost":2}
  ]
}`

const stubAPILimits = `{"usage_tier":"advanced",
  "read":{"refill_rate":400,"bucket_capacity":800},
  "write":{"refill_rate":200,"bucket_capacity":400}}`

// configuredClient returns a client whose cost table came through
// ConfigureRateLimits, exactly as it does in production.
func configuredClient(t *testing.T, costsBody string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/trade-api/v2/account/limits":
			fmt.Fprint(w, stubAPILimits)
		case "/trade-api/v2/account/endpoint_costs":
			fmt.Fprint(w, costsBody)
		default:
			t.Errorf("unexpected request during configuration: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)
	require.NoError(t, c.ConfigureRateLimits(context.Background()))
	return c
}

func TestConfigureRateLimits_ResolvesParameterizedPaths(t *testing.T) {
	c := configuredClient(t, liveEndpointCostsPayload)

	tests := []struct {
		name     string
		method   string
		path     string
		wantCost float64
	}{
		{
			name:   "cancel order — the highest-frequency write",
			method: http.MethodDelete,
			path:   "/trade-api/v2/portfolio/events/orders/01a097c2-ef50-7aeb-afd8-82e775ead250",
			// Billed 10 before this fix: the ":order_id" entry never matched.
			wantCost: 2,
		},
		{
			name:     "batched beats the parameterized sibling",
			method:   http.MethodDelete,
			path:     "/trade-api/v2/portfolio/events/orders/batched",
			wantCost: 2,
		},
		{
			name:     "cancel all orders — no placeholder",
			method:   http.MethodDelete,
			path:     "/trade-api/v2/portfolio/events/orders",
			wantCost: 2,
		},
		{
			name:     "two placeholders in one path",
			method:   http.MethodDelete,
			path:     "/trade-api/v2/communications/rfqs/rfq-1/quotes/quote-1",
			wantCost: 2,
		},
		{
			name:     "placeholders followed by a literal",
			method:   http.MethodPut,
			path:     "/trade-api/v2/communications/rfqs/rfq-1/quotes/quote-1/confirm",
			wantCost: 1,
		},
		{
			name:     "trailing wildcard consumes the remaining segments",
			method:   http.MethodGet,
			path:     "/trade-api/v2/cfbenchmarks/some/nested/thing",
			wantCost: 50,
		},
		{
			name:     "unlisted endpoint falls back to the default",
			method:   http.MethodGet,
			path:     "/trade-api/v2/markets",
			wantCost: 10,
		},
		{
			name:   "a placeholder matches one segment, not several",
			method: http.MethodGet,
			// No entry covers this, so it must fall back rather than be
			// claimed by /communications/quotes/:quote_id.
			path:     "/trade-api/v2/communications/quotes/q-1/extra",
			wantCost: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			read, write := c.resolveCosts(tt.method, tt.path, 999, 999, 1)
			got := write
			if tt.method == http.MethodGet {
				got = read
			}
			assert.Equal(t, tt.wantCost, got)
		})
	}
}

// Every entry the API returns must match a real request path. If Kalshi changes
// its placeholder syntax again, this fails instead of silently reverting those
// endpoints to the default cost.
func TestConfigureRateLimits_EveryEntryIsReachable(t *testing.T) {
	c := configuredClient(t, liveEndpointCostsPayload)

	require.NotEmpty(t, c.costRoutes)
	for _, route := range c.costRoutes {
		// Substitute a plausible value for each placeholder to build a path
		// the API would actually receive.
		concrete := make([]string, 0, len(route.segments))
		for _, seg := range route.segments {
			if isPathPlaceholder(seg) {
				concrete = append(concrete, "concrete-value")
				continue
			}
			concrete = append(concrete, seg)
		}
		path := "/" + joinSegments(concrete)

		cost, matched := c.lookupCost(route.method, path)
		assert.True(t, matched,
			"%s %s produced a route that matches no request path", route.method, route.rawPath)
		assert.Equal(t, route.cost, cost,
			"%s %s resolved to another route's cost", route.method, route.rawPath)
	}
}

// Batch endpoints are billed per item and the table cannot express that, so the
// caller's count must survive. Before this fix the resolved cost replaced the
// total and a fifty-order batch was billed as a single request.
func TestResolveCosts_BatchBillsPerItem(t *testing.T) {
	c := configuredClient(t, liveEndpointCostsPayload)

	t.Run("batch cancel multiplies the table cost", func(t *testing.T) {
		_, write := c.resolveCosts(http.MethodDelete,
			"/trade-api/v2/portfolio/events/orders/batched", 0, costCancelOrder, 50)
		assert.Equal(t, 100.0, write, "50 cancels at 2 tokens each")
	})

	t.Run("batch create multiplies the default cost", func(t *testing.T) {
		// Absent from the table, because its per-item cost equals the default —
		// so the multiplier is the only thing that makes it correct.
		_, write := c.resolveCosts(http.MethodPost,
			"/trade-api/v2/portfolio/events/orders/batched", 0, costCreateOrder, 50)
		assert.Equal(t, 500.0, write, "50 creates at the default 10 tokens each")
	})

	t.Run("a single request is unscaled", func(t *testing.T) {
		_, write := c.resolveCosts(http.MethodDelete,
			"/trade-api/v2/portfolio/events/orders/abc", 0, costCancelOrder, 1)
		assert.Equal(t, 2.0, write)
	})

	t.Run("zero units is treated as one", func(t *testing.T) {
		_, write := c.resolveCosts(http.MethodDelete,
			"/trade-api/v2/portfolio/events/orders/batched", 0, costCancelOrder, 0)
		assert.Equal(t, 2.0, write, "an empty batch still costs one request")
	})
}

// Without a table the caller's literals are authoritative, and they must still
// account for units.
func TestResolveCosts_NoTableUsesCallerCost(t *testing.T) {
	c := &Client{}
	read, write := c.resolveCosts(http.MethodPost, "/trade-api/v2/anything", 0, 40, 4)
	assert.Equal(t, 0.0, read)
	assert.Equal(t, 40.0, write, "the caller's figure is used as given")
}

func joinSegments(segments []string) string {
	out := ""
	for i, s := range segments {
		if i > 0 {
			out += "/"
		}
		out += s
	}
	return out
}

// If Kalshi ever serves path parameters without a marker, the entry would be
// treated as a literal and quietly stop matching. That is the failure this
// package already had, so it must announce itself.
func TestConfigureRateLimits_FlagsUnrecognisedPlaceholders(t *testing.T) {
	tests := []struct {
		segment string
		suspect bool
	}{
		{":order_id", false},  // recognised
		{"{order_id}", false}, // recognised
		{"*endpoint", false},  // recognised
		{"order_id", true},    // unmarked parameter
		{"id", true},
		{"orders", false}, // ordinary literal
		{"batched", false},
		{"events", false},
	}
	for _, tt := range tests {
		t.Run(tt.segment, func(t *testing.T) {
			assert.Equal(t, tt.suspect, suspectUnmarkedPlaceholder(tt.segment))
		})
	}
}
