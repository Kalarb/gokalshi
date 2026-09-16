package gokalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
				return c.CancelAllOrders(context.Background(), CancelAllOrdersParams{Subaccount: intPtr(7)})
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

// --- Wire-level contract tests -------------------------------------------
//
// The tests above assert method and path. These assert what actually goes on
// the wire and what comes back off it, which is where serialization defects
// live: a field dropped by omitempty looks identical to a field the caller
// never set.

// decodeBody returns the JSON body the client sent.
func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	return body
}

// CancelAllOrders cancels across every shard and every subaccount when
// subaccount is omitted. A caller naming subaccount 0 means "the primary
// subaccount only" and must not silently get the account-wide behaviour.
func TestCancelAllOrders_SubaccountIsSent(t *testing.T) {
	tests := []struct {
		name       string
		subaccount *int
		want       string
	}{
		{"unset omits the parameter", nil, ""},
		{"zero scopes to the primary subaccount", intPtr(0), "0"},
		{"non-zero scopes to that subaccount", intPtr(4), "4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.want, r.URL.Query().Get("subaccount"))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			c := newTestClient(t, srv.URL)
			require.NoError(t, c.CancelAllOrders(context.Background(),
				CancelAllOrdersParams{Subaccount: tt.subaccount}))
		})
	}
}

// exchange_index 0 is the event-contract shard. Omitting it means auto-route,
// which bills every shard's write bucket rather than one, so the two are not
// interchangeable.
func TestCreateOrderV2_ExchangeIndexIsSent(t *testing.T) {
	tests := []struct {
		name    string
		index   *int
		present bool
		want    float64
	}{
		{"unset omits the field", nil, false, 0},
		{"zero pins to the event-contract shard", intPtr(0), true, 0},
		{"minus one requests auto-routing", intPtr(-1), true, -1},
		{"positive pins to that shard", intPtr(2), true, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := decodeBody(t, r)
				got, ok := body["exchange_index"]
				assert.Equal(t, tt.present, ok, "exchange_index present in body")
				if tt.present {
					assert.Equal(t, tt.want, got)
				}
				fmt.Fprint(w, `{"order_id":"o-1"}`)
			}))
			defer srv.Close()

			c := newTestClient(t, srv.URL)
			req := newTestOrderV2()
			req.ExchangeIndex = tt.index
			_, err := c.CreateOrderV2(context.Background(), req)
			require.NoError(t, err)
		})
	}
}

// An empty allocations array is how the API is told to disable automatic
// rebalancing. JSON null is a different request and must not be produced by
// an explicitly empty slice.
func TestSetTargetBalanceAllocation_EmptyAllocationsIsArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"allocations":[]`,
			"an empty slice must serialize as [] — null does not disable rebalancing")
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.SetTargetBalanceAllocation(context.Background(), SetTargetBalanceAllocationRequest{
		Allocations: []TargetBalanceAllocationInput{},
	})
	require.NoError(t, err)
}

// newTestOrderV2 is a minimal valid V2 order for wire-level assertions.
func newTestOrderV2() CreateOrderV2Request {
	return CreateOrderV2Request{
		Ticker:                  "KXTEST-1",
		Side:                    BookSideBid,
		Count:                   "1.00",
		Price:                   "0.0100",
		TimeInForce:             TimeInForceGTC,
		SelfTradePreventionType: STPTakerAtCross,
	}
}

// TestNewEndpoints_RequestBodies asserts the JSON each endpoint actually puts
// on the wire. Without this, a field renamed by the generator or dropped by
// omitempty is indistinguishable from one the caller never set.
func TestNewEndpoints_RequestBodies(t *testing.T) {
	tests := []struct {
		name string
		want map[string]any
		call func(*Client) error
	}{
		{
			name: "IntraExchangeInstanceTransfer sends amount in centicents",
			want: map[string]any{
				"source":                 "event_contract",
				"destination":            "event_contract",
				"amount":                 float64(100),
				"destination_subaccount": float64(1),
			},
			call: func(c *Client) error {
				_, err := c.IntraExchangeInstanceTransfer(context.Background(),
					IntraExchangeInstanceTransferRequest{
						Source:                ExchangeInstanceEventContract,
						Destination:           ExchangeInstanceEventContract,
						Amount:                100,
						DestinationSubaccount: 1,
					})
				return err
			},
		},
		{
			name: "SetTargetBalanceAllocation sends allocations and reservation",
			want: map[string]any{
				"allocations": []any{
					map[string]any{"exchange_index": float64(0), "percent": float64(60)},
					map[string]any{"exchange_index": float64(1), "percent": float64(40)},
				},
				"resting_margin_reservation": "sum",
			},
			call: func(c *Client) error {
				_, err := c.SetTargetBalanceAllocation(context.Background(),
					SetTargetBalanceAllocationRequest{
						Allocations: []TargetBalanceAllocationInput{
							{ExchangeIndex: 0, Percent: 60},
							{ExchangeIndex: 1, Percent: 40},
						},
						RestingMarginReservation: RestingMarginReservationSum,
					})
				return err
			},
		},
		{
			name: "ProposeBlockTrade sends centi-denominated price and count",
			want: map[string]any{
				"buyer_user_id":     "u-buy",
				"seller_user_id":    "u-sell",
				"market_ticker":     "KXTEST-1",
				"price_centi_cents": float64(5000),
				"centicount":        float64(1000),
				"maker_side":        "yes",
				"expiration_ts":     "2026-09-13T00:00:00Z",
			},
			call: func(c *Client) error {
				_, err := c.ProposeBlockTrade(context.Background(), ProposeBlockTradeRequest{
					BuyerUserID: "u-buy", SellerUserID: "u-sell", MarketTicker: "KXTEST-1",
					PriceCentiCents: 5000, Centicount: 1000, MakerSide: "yes",
					ExpirationTS: "2026-09-13T00:00:00Z",
				})
				return err
			},
		},
		{
			name: "AcceptRFQQuote sends the accepted side",
			want: map[string]any{"accepted_side": "yes"},
			call: func(c *Client) error {
				return c.AcceptRFQQuote(context.Background(), "r-1", "q-1",
					AcceptQuoteRequest{AcceptedSide: SideYes})
			},
		},
		{
			name: "AcceptBlockTradeProposal sends the subtrader id",
			want: map[string]any{"subtrader_id": "u_1"},
			call: func(c *Client) error {
				return c.AcceptBlockTradeProposal(context.Background(), "bt-1",
					AcceptBlockTradeProposalRequest{SubtraderID: "u_1"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := decodeBody(t, r)
				for k, want := range tt.want {
					assert.Equal(t, want, got[k], "field %q", k)
				}
				fmt.Fprint(w, `{"transfer_id":"t-1","block_trade_proposal_id":"bt-1"}`)
			}))
			defer srv.Close()

			require.NoError(t, tt.call(newTestClient(t, srv.URL)))
		})
	}
}

// TestNewEndpoints_ResponseDecoding asserts responses land in the typed models.
// The path tests above return `{}`, so nothing there would notice a renamed or
// wrongly-typed field.
func TestNewEndpoints_ResponseDecoding(t *testing.T) {
	t.Run("GetWeatherIndex", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{
			"city":"miami","units":"fahrenheit","config_version":"miami-temperature-v1.0",
			"timeseries":[{"t":1757700000000,"status":"normal","v":88.25,"contributors":3}]
		}`))
		resp, err := c.GetWeatherIndex(context.Background(), "miami", GetWeatherIndexParams{})
		require.NoError(t, err)
		assert.Equal(t, "miami", resp.City)
		assert.Equal(t, "fahrenheit", resp.Units)
		assert.Equal(t, "miami-temperature-v1.0", resp.ConfigVersion)
		require.Len(t, resp.Timeseries, 1)
		assert.Equal(t, int64(1757700000000), resp.Timeseries[0].T)
		assert.InDelta(t, 88.25, resp.Timeseries[0].V, 0.001)
		assert.Equal(t, 3, resp.Timeseries[0].Contributors)
	})

	t.Run("GetWeatherIndexCalibrations", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{
			"city":"miami","units":"celsius",
			"calibrations":[{"config_version":"miami-temperature-v1.0","effective_at_ms":1756600000000,
			                 "city_reference_c":24.5,"stations":[]}]
		}`))
		resp, err := c.GetWeatherIndexCalibrations(context.Background(), "miami")
		require.NoError(t, err)
		// Units differ from GetWeatherIndex by design: the published index is
		// Fahrenheit, the calibration offsets are Celsius.
		assert.Equal(t, "celsius", resp.Units)
		require.Len(t, resp.Calibrations, 1)
		assert.Equal(t, "miami-temperature-v1.0", resp.Calibrations[0].ConfigVersion)
		assert.InDelta(t, 24.5, resp.Calibrations[0].CityReferenceC, 0.001)
	})

	t.Run("GetIntraExchangeInstanceTransfer", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{"transfer":{
			"transfer_id":"t-1","source":"event_contract","destination":"margined",
			"source_exchange_shard":0,"destination_exchange_shard":1,
			"amount":"0.0100","status":"complete","created_ts":1757700000}}`))
		resp, err := c.GetIntraExchangeInstanceTransfer(context.Background(), "t-1")
		require.NoError(t, err)
		assert.Equal(t, "t-1", resp.Transfer.TransferID)
		assert.Equal(t, ExchangeInstanceMargined, resp.Transfer.Destination)
		assert.Equal(t, 1, resp.Transfer.DestinationExchangeShard)
		// The request takes int64 centicents; the response is a dollar string.
		assert.Equal(t, "0.0100", resp.Transfer.Amount)
		assert.Equal(t, IntraExchangeInstanceTransferStatusComplete, resp.Transfer.Status)
	})

	t.Run("GetTargetBalanceAllocation", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{
			"allocations":[{"exchange_index":0,"percent":70},{"exchange_index":1,"percent":30}],
			"resting_margin_reservation":"max"}`))
		resp, err := c.GetTargetBalanceAllocation(context.Background())
		require.NoError(t, err)
		assert.Equal(t, RestingMarginReservationMax, resp.RestingMarginReservation)
		require.Len(t, resp.Allocations, 2)
		total := 0
		for _, a := range resp.Allocations {
			total += a.Percent
		}
		assert.Equal(t, 100, total, "allocation percentages must total 100")
	})

	t.Run("GetEventLiveData", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{"live_data":{
			"type":"crypto_price","default_range":"1h","range_options":["15min","1h","1d"],
			"is_historical":true,"details":{"symbol":"BTC"}}}`))
		resp, err := c.GetEventLiveData(context.Background(), "KXEVENT", GetEventLiveDataParams{})
		require.NoError(t, err)
		assert.Equal(t, "crypto_price", resp.LiveData.Type)
		assert.Equal(t, "1h", resp.LiveData.DefaultRange)
		assert.Equal(t, []string{"15min", "1h", "1d"}, resp.LiveData.RangeOptions)
		assert.True(t, resp.LiveData.IsHistorical)
		// details is deliberately untyped — its shape depends on type.
		assert.Equal(t, "BTC", resp.LiveData.Details["symbol"])
	})

	t.Run("GetAccountAPIUsageLevelVolumeProgress", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{"volume_progress":[{
			"computed_ts":1757700000,"trailing_30d_volume_fp":"12345.00",
			"goals":[{"level":"Expert","earn_volume_goal_fp":"50000.00","keep_volume_goal_fp":"25000.00"}]}]}`))
		resp, err := c.GetAccountAPIUsageLevelVolumeProgress(context.Background())
		require.NoError(t, err)
		require.Len(t, resp.VolumeProgress, 1)
		assert.Equal(t, "12345.00", resp.VolumeProgress[0].Trailing30dVolumeFP)
		require.Len(t, resp.VolumeProgress[0].Goals, 1)
		assert.Equal(t, "Expert", resp.VolumeProgress[0].Goals[0].Level)
	})

	t.Run("GetBlockTradeProposals", func(t *testing.T) {
		c := newTestClient(t, stubServer(t, `{"block_trade_proposals":[{
			"id":"bt-1","proposer_user_id":"u-1","buyer_user_id":"u-1","seller_user_id":"",
			"market_ticker":"KXTEST-1","price_centi_cents":5000,"centicount":1000,
			"maker_side":"yes","expiration_ts":"2026-09-13T00:00:00Z","status":"pending",
			"created_ts":"2026-09-12T00:00:00Z","updated_ts":"2026-09-12T00:00:00Z",
			"buyer_accepted":true,"seller_accepted":false}]}`))
		resp, err := c.GetBlockTradeProposals(context.Background(), GetBlockTradeProposalsParams{})
		require.NoError(t, err)
		require.Len(t, resp.BlockTradeProposals, 1)
		p := resp.BlockTradeProposals[0]
		assert.Equal(t, int64(5000), p.PriceCentiCents)
		assert.True(t, p.BuyerAccepted)
		assert.False(t, p.SellerAccepted)
		// seller_user_id is empty when the caller is not the seller — required
		// by the spec but legitimately blank.
		assert.Empty(t, p.SellerUserID)
	})
}

// stubServer returns a server that answers every request with body, closed on
// test cleanup.
func stubServer(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}
