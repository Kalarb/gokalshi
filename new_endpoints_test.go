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

// Covers the endpoints added for 3.30.0 parity. Each case asserts the method,
// the path and any query or body the caller cannot otherwise observe.

func TestNewEndpoints_PathsAndMethods(t *testing.T) {
	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantQuery  map[string]string
		body       string
		call       func(*Client) error
	}{
		{
			name: "CancelAllOrders", wantMethod: http.MethodDelete,
			wantPath: "/trade-api/v2/portfolio/events/orders",
			body:     "", // 204, no body
			call: func(c *Client) error {
				return c.CancelAllOrders(context.Background(), CancelAllOrdersParams{})
			},
		},
		{
			name: "CancelAllOrders scoped to a subaccount", wantMethod: http.MethodDelete,
			wantPath:  "/trade-api/v2/portfolio/events/orders",
			wantQuery: map[string]string{"subaccount": "7"},
			call: func(c *Client) error {
				return c.CancelAllOrders(context.Background(), CancelAllOrdersParams{Subaccount: 7})
			},
		},
		{
			name: "GetAccountAPIUsageLevelVolumeProgress", wantMethod: http.MethodGet,
			wantPath: "/trade-api/v2/account/api_usage_level/volume_progress",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.GetAccountAPIUsageLevelVolumeProgress(context.Background())
				return err
			},
		},
		{
			name: "GetHistoricalPositions", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/historical/positions",
			wantQuery: map[string]string{"ticker": "KX", "limit": "25"},
			body:      `{"market_positions":[],"event_positions":[]}`,
			call: func(c *Client) error {
				_, err := c.GetHistoricalPositions(context.Background(),
					GetHistoricalPositionsParams{Ticker: "KX", Limit: 25})
				return err
			},
		},
		{
			name: "GetEventLiveData", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/live_data/events/KXEVENT",
			wantQuery: map[string]string{"range": "1h"},
			body:      `{}`,
			call: func(c *Client) error {
				_, err := c.GetEventLiveData(context.Background(), "KXEVENT",
					GetEventLiveDataParams{Range: "1h"})
				return err
			},
		},
		{
			name: "GetWeatherIndex", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/live_data/weather/nyc",
			wantQuery: map[string]string{"detailed": "true", "last_sec": "600"},
			body:      `{}`,
			call: func(c *Client) error {
				_, err := c.GetWeatherIndex(context.Background(), "nyc",
					GetWeatherIndexParams{Detailed: true, LastSec: 600})
				return err
			},
		},
		{
			name: "GetWeatherIndexCalibrations", wantMethod: http.MethodGet,
			wantPath: "/trade-api/v2/live_data/weather/nyc/calibrations",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.GetWeatherIndexCalibrations(context.Background(), "nyc")
				return err
			},
		},
		{
			name: "GetTargetBalanceAllocation", wantMethod: http.MethodGet,
			wantPath: "/trade-api/v2/portfolio/target_balance_allocation",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.GetTargetBalanceAllocation(context.Background())
				return err
			},
		},
		{
			name: "SetTargetBalanceAllocation", wantMethod: http.MethodPost,
			wantPath: "/trade-api/v2/portfolio/target_balance_allocation",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.SetTargetBalanceAllocation(context.Background(),
					SetTargetBalanceAllocationRequest{})
				return err
			},
		},
		{
			name: "IntraExchangeInstanceTransfer", wantMethod: http.MethodPost,
			wantPath: "/trade-api/v2/portfolio/intra_exchange_instance_transfer",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.IntraExchangeInstanceTransfer(context.Background(),
					IntraExchangeInstanceTransferRequest{})
				return err
			},
		},
		{
			name: "GetIntraExchangeInstanceTransfers", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/portfolio/intra_exchange_instance_transfers",
			wantQuery: map[string]string{"limit": "10"},
			body:      `{}`,
			call: func(c *Client) error {
				_, err := c.GetIntraExchangeInstanceTransfers(context.Background(),
					GetIntraExchangeInstanceTransfersParams{Limit: 10})
				return err
			},
		},
		{
			name: "GetIntraExchangeInstanceTransfer", wantMethod: http.MethodGet,
			wantPath: "/trade-api/v2/portfolio/intra_exchange_instance_transfers/t-1",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.GetIntraExchangeInstanceTransfer(context.Background(), "t-1")
				return err
			},
		},
		{
			name: "GetBlockTradeProposals", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/communications/block-trade-proposals",
			wantQuery: map[string]string{"status": "pending"},
			body:      `{}`,
			call: func(c *Client) error {
				_, err := c.GetBlockTradeProposals(context.Background(),
					GetBlockTradeProposalsParams{Status: "pending"})
				return err
			},
		},
		{
			name: "ProposeBlockTrade", wantMethod: http.MethodPost,
			wantPath: "/trade-api/v2/communications/block-trade-proposals",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.ProposeBlockTrade(context.Background(), ProposeBlockTradeRequest{})
				return err
			},
		},
		{
			name: "AcceptBlockTradeProposal", wantMethod: http.MethodPost,
			wantPath: "/trade-api/v2/communications/block-trade-proposals/bt-1/accept",
			call: func(c *Client) error {
				return c.AcceptBlockTradeProposal(context.Background(), "bt-1",
					AcceptBlockTradeProposalRequest{})
			},
		},
		{
			name: "GetRFQQuote", wantMethod: http.MethodGet,
			wantPath: "/trade-api/v2/communications/rfqs/r-1/quotes/q-1",
			body:     `{}`,
			call: func(c *Client) error {
				_, err := c.GetRFQQuote(context.Background(), "r-1", "q-1")
				return err
			},
		},
		{
			name: "DeleteRFQQuote", wantMethod: http.MethodDelete,
			wantPath: "/trade-api/v2/communications/rfqs/r-1/quotes/q-1",
			call: func(c *Client) error {
				return c.DeleteRFQQuote(context.Background(), "r-1", "q-1")
			},
		},
		{
			name: "AcceptRFQQuote", wantMethod: http.MethodPut,
			wantPath: "/trade-api/v2/communications/rfqs/r-1/quotes/q-1/accept",
			call: func(c *Client) error {
				return c.AcceptRFQQuote(context.Background(), "r-1", "q-1", AcceptQuoteRequest{})
			},
		},
		{
			name: "ConfirmRFQQuote", wantMethod: http.MethodPut,
			wantPath: "/trade-api/v2/communications/rfqs/r-1/quotes/q-1/confirm",
			call: func(c *Client) error {
				return c.ConfirmRFQQuote(context.Background(), "r-1", "q-1")
			},
		},
		{
			name: "GetFCMOrders", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/fcm/orders",
			wantQuery: map[string]string{"subtrader_id": "u_1"},
			body:      `{"orders":[]}`,
			call: func(c *Client) error {
				_, err := c.GetFCMOrders(context.Background(), GetFCMOrdersParams{SubtraderID: "u_1"})
				return err
			},
		},
		{
			name: "GetFCMPositions", wantMethod: http.MethodGet,
			wantPath:  "/trade-api/v2/fcm/positions",
			wantQuery: map[string]string{"subtrader_id": "u_1"},
			body:      `{"market_positions":[],"event_positions":[]}`,
			call: func(c *Client) error {
				_, err := c.GetFCMPositions(context.Background(), GetFCMPositionsParams{SubtraderID: "u_1"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantMethod, r.Method)
				assert.Equal(t, tt.wantPath, r.URL.Path)
				for k, v := range tt.wantQuery {
					assert.Equal(t, v, r.URL.Query().Get(k), "query param %q", k)
				}
				if tt.body == "" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			require.NoError(t, tt.call(newTestClient(t, srv.URL)))
		})
	}
}

// exchange_index carries meaning at 0 and -1, so it must survive as a query
// parameter rather than being filtered out as a zero value.
func TestCancelOrderV2Params_ExchangeIndexIsSent(t *testing.T) {
	tests := []struct {
		name  string
		value *int
		want  string
	}{
		{"unset is omitted", nil, ""},
		{"zero selects event-contract", intPtr(0), "0"},
		{"minus one auto-routes", intPtr(-1), "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.want, r.URL.Query().Get("exchange_index"))
				fmt.Fprint(w, `{"order_id":"o-1"}`)
			}))
			defer srv.Close()

			c := newTestClient(t, srv.URL)
			_, err := c.CancelOrderV2(context.Background(), "o-1",
				CancelOrderV2Params{ExchangeIndex: tt.value})
			require.NoError(t, err)
		})
	}
}
