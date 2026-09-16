//go:build integration

package gokalshi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPIntegration_Historical(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	marketsResp, err := c.GetHistoricalMarkets(ctx, GetHistoricalMarketsParams{Limit: 5})
	require.NoError(t, err)
	var historicalTicker string
	if len(marketsResp.Markets) > 0 {
		historicalTicker = marketsResp.Markets[0].Ticker
	}
	t.Logf("found %d historical markets", len(marketsResp.Markets))

	t.Run("GetHistoricalCutoff", func(t *testing.T) {
		resp, err := c.GetHistoricalCutoff(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.MarketSettledTS)
		t.Logf("cutoff market_settled=%s orders_updated=%s trades_created=%s",
			resp.MarketSettledTS, resp.OrdersUpdatedTS, resp.TradesCreatedTS)
	})

	t.Run("GetHistoricalFills", func(t *testing.T) {
		_, err := c.GetHistoricalFills(ctx, GetHistoricalFillsParams{Limit: 5})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetHistoricalOrders", func(t *testing.T) {
		_, err := c.GetHistoricalOrders(ctx, GetHistoricalOrdersParams{Limit: 5})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetHistoricalTrades", func(t *testing.T) {
		_, err := c.GetHistoricalTrades(ctx, GetHistoricalTradesParams{Limit: 5})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetHistoricalMarkets", func(t *testing.T) {
		assert.NotNil(t, marketsResp.Markets)
		t.Logf("historical markets count=%d", len(marketsResp.Markets))
	})

	t.Run("GetHistoricalMarket", func(t *testing.T) {
		if historicalTicker == "" {
			t.Skip("no historical markets available")
		}
		resp, err := c.GetHistoricalMarket(ctx, historicalTicker)
		require.NoError(t, err)
		assert.Equal(t, historicalTicker, resp.Market.Ticker)
	})

	t.Run("GetHistoricalMarketCandlesticks", func(t *testing.T) {
		if historicalTicker == "" {
			t.Skip("no historical markets available")
		}
		now := time.Now().Unix()
		resp, err := c.GetHistoricalMarketCandlesticks(ctx, historicalTicker, GetHistoricalMarketCandlesticksParams{
			StartTS:        now - 86400*30,
			EndTS:          now,
			PeriodInterval: 1440,
		})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("historical candlesticks: ticker=%s count=%d", resp.Ticker, len(resp.Candlesticks))
	})
}

func TestHTTPIntegration_MilestonesAndLiveData(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	milestonesResp, err := c.GetMilestones(ctx, GetMilestonesParams{Limit: 5})
	require.NoError(t, err)
	if len(milestonesResp.Milestones) == 0 {
		t.Skip("no milestones available")
	}
	milestoneID := milestonesResp.Milestones[0].ID
	t.Logf("found %d milestones, using %s", len(milestonesResp.Milestones), milestoneID)

	t.Run("GetMilestones", func(t *testing.T) {
		assert.NotEmpty(t, milestonesResp.Milestones)
	})

	t.Run("GetMilestone", func(t *testing.T) {
		resp, err := c.GetMilestone(ctx, milestoneID)
		require.NoError(t, err)
		assert.Equal(t, milestoneID, resp.Milestone.ID)
	})

	t.Run("GetLiveDataByMilestone", func(t *testing.T) {
		_, err := c.GetLiveDataByMilestone(ctx, milestoneID, GetLiveDataParams{})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetLiveDataBatch", func(t *testing.T) {
		var ids string
		for i, m := range milestonesResp.Milestones {
			if i > 0 {
				ids += ","
			}
			ids += m.ID
			if i >= 2 {
				break
			}
		}
		_, err := c.GetLiveDataBatch(ctx, GetLiveDataBatchParams{MilestoneIDs: ids})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetMilestoneGameStats", func(t *testing.T) {
		_, err := c.GetMilestoneGameStats(ctx, milestoneID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("GetLiveData", func(t *testing.T) {
		_, err := c.GetLiveData(ctx, "game", milestoneID, GetLiveDataParams{})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})
}

func TestHTTPIntegration_MVECollections(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	collectionsResp, err := c.GetMultivariateEventCollections(ctx, GetMultivariateEventCollectionsParams{Limit: 5})
	require.NoError(t, err)
	if len(collectionsResp.MultivariateContracts) == 0 {
		t.Skip("no multivariate event collections available")
	}
	collectionTicker := collectionsResp.MultivariateContracts[0].CollectionTicker
	t.Logf("found %d MVE collections, using %s", len(collectionsResp.MultivariateContracts), collectionTicker)

	t.Run("GetMultivariateEventCollections", func(t *testing.T) {
		assert.NotEmpty(t, collectionsResp.MultivariateContracts)
	})

	t.Run("GetMultivariateEventCollection", func(t *testing.T) {
		resp, err := c.GetMultivariateEventCollection(ctx, collectionTicker)
		require.NoError(t, err)
		assert.Equal(t, collectionTicker, resp.MultivariateContract.CollectionTicker)
	})
}

func TestHTTPIntegration_StructuredTargets(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	targetsResp, err := c.GetStructuredTargets(ctx, GetStructuredTargetsParams{Limit: 5})
	require.NoError(t, err)
	if len(targetsResp.StructuredTargets) == 0 {
		t.Skip("no structured targets available")
	}
	targetID := targetsResp.StructuredTargets[0].ID
	t.Logf("found %d structured targets, using %s", len(targetsResp.StructuredTargets), targetID)

	t.Run("GetStructuredTargets", func(t *testing.T) {
		assert.NotEmpty(t, targetsResp.StructuredTargets)
	})

	t.Run("GetStructuredTarget", func(t *testing.T) {
		resp, err := c.GetStructuredTarget(ctx, targetID)
		require.NoError(t, err)
		assert.Equal(t, targetID, resp.StructuredTarget.ID)
	})
}

// ---------------------------------------------------------------------------
// Endpoints added for OpenAPI 3.30.0 parity — read paths
// ---------------------------------------------------------------------------

func TestHTTPIntegration_WeatherIndex(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	city := weatherCity(t, c, ctx)

	// Fetch calibrations first: the index response names a config_version, and
	// the only way to know it is a real one is to find it in this list.
	calibrations, err := c.GetWeatherIndexCalibrations(ctx, city)
	skipOnAPIError(t, err, 400, 404)
	require.NoError(t, err)

	t.Run("GetWeatherIndexCalibrations", func(t *testing.T) {
		assert.Equal(t, city, calibrations.City)
		// Celsius here is deliberate — the published index is Fahrenheit, but
		// calibration offsets are Celsius quantities from the methodology.
		assert.Equal(t, "celsius", calibrations.Units)
		require.NotEmpty(t, calibrations.Calibrations, "a configured city always has a launch calibration")

		// The spec states the timeline is append-only, complete, and ascending
		// by effective time.
		for i := 1; i < len(calibrations.Calibrations); i++ {
			assert.GreaterOrEqual(t,
				calibrations.Calibrations[i].EffectiveAtMs,
				calibrations.Calibrations[i-1].EffectiveAtMs,
				"calibrations must be ascending by effective time")
		}
		assert.Zero(t, calibrations.Calibrations[0].CalibrationWindowStartMs,
			"the first record is the launch configuration and is not derived from a window")
		t.Logf("city=%s calibrations=%d", city, len(calibrations.Calibrations))
	})

	t.Run("GetWeatherIndex", func(t *testing.T) {
		resp, err := c.GetWeatherIndex(ctx, city, GetWeatherIndexParams{LastSec: 3600})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)

		assert.Equal(t, city, resp.City)
		assert.Equal(t, "fahrenheit", resp.Units)

		if resp.ConfigVersion == "" {
			t.Skip("no points matched the window, so config_version is empty")
		}
		known := make([]string, 0, len(calibrations.Calibrations))
		for _, cal := range calibrations.Calibrations {
			known = append(known, cal.ConfigVersion)
		}
		assert.Contains(t, known, resp.ConfigVersion,
			"the index config_version must name a published calibration")

		// Gaps are real: minutes where the index quorum failed are omitted
		// entirely, so only assert on the points that came back.
		for _, p := range resp.Timeseries {
			assert.NotZero(t, p.T, "every point carries an event minute")
			assert.NotEmpty(t, p.Status)
		}
		t.Logf("points=%d config_version=%s", len(resp.Timeseries), resp.ConfigVersion)
	})

	t.Run("GetWeatherIndexUnknownCity", func(t *testing.T) {
		_, err := c.GetWeatherIndex(ctx, "not-a-real-city", GetWeatherIndexParams{})
		require.Error(t, err)
		// The spec documents 400 for an unknown city, not 404.
		assert.True(t, isAPIErrorCode(err, 400),
			"unknown city should be a structured 400, got: %v", err)
	})
}

func TestHTTPIntegration_EventLiveData(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	eventTicker := liveDataEvent(t, c, ctx)

	// Self-describing round-trip: the first response advertises which ranges it
	// supports, so the second call is driven by the API rather than a hardcoded
	// guess that rots when the event type changes.
	first, err := c.GetEventLiveData(ctx, eventTicker, GetEventLiveDataParams{})
	skipOnAPIError(t, err, 400, 404)
	require.NoError(t, err)

	t.Run("GetEventLiveData", func(t *testing.T) {
		assert.NotEmpty(t, first.LiveData.Type, "type names the schema of details")
		assert.NotNil(t, first.LiveData.Details)
		t.Logf("event=%s type=%s default_range=%q options=%v",
			eventTicker, first.LiveData.Type, first.LiveData.DefaultRange, first.LiveData.RangeOptions)

		if first.LiveData.DefaultRange == "" {
			t.Skip("live data type does not advertise a default range")
		}
		assert.Contains(t, first.LiveData.RangeOptions, first.LiveData.DefaultRange,
			"the default range must be one of the offered options")

		second, err := c.GetEventLiveData(ctx, eventTicker,
			GetEventLiveDataParams{Range: first.LiveData.DefaultRange})
		require.NoError(t, err)
		assert.Equal(t, first.LiveData.Type, second.LiveData.Type,
			"restricting the window must not change the data type")
	})
}

func TestHTTPIntegration_HistoricalPositions(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	cutoff, err := c.GetHistoricalCutoff(ctx)
	require.NoError(t, err)
	t.Logf("market_positions_last_updated_ts=%s", cutoff.MarketPositionsLastUpdatedTS)

	t.Run("GetHistoricalPositions", func(t *testing.T) {
		historical, err := c.GetHistoricalPositions(ctx, GetHistoricalPositionsParams{Limit: 50})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotNil(t, historical.MarketPositions)
		t.Logf("historical positions=%d", len(historical.MarketPositions))

		if len(historical.MarketPositions) == 0 {
			t.Skip("no archived positions on this account")
		}

		// The spec guarantees a clean partition: a settled event's positions
		// move here together and are never split with /portfolio/positions.
		live, err := c.GetPositions(ctx, GetPositionsParams{Limit: 200})
		require.NoError(t, err)
		liveTickers := map[string]bool{}
		for _, p := range live.MarketPositions {
			liveTickers[p.Ticker] = true
		}
		for _, p := range historical.MarketPositions {
			assert.False(t, liveTickers[p.Ticker],
				"ticker %s appears in both historical and live positions", p.Ticker)
		}
	})
}

func TestHTTPIntegration_AccountUsageAndAllocation(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetAccountAPIUsageLevelVolumeProgress", func(t *testing.T) {
		resp, err := c.GetAccountAPIUsageLevelVolumeProgress(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		// Cron-computed, so an empty list is legitimate on a quiet account.
		assert.NotNil(t, resp.VolumeProgress)
		for _, p := range resp.VolumeProgress {
			assert.NotEmpty(t, p.Trailing30dVolumeFP)
			for _, g := range p.Goals {
				assert.NotEmpty(t, g.Level, "every volume goal names its tier")
			}
		}
		t.Logf("volume progress entries=%d", len(resp.VolumeProgress))
	})

	t.Run("GetTargetBalanceAllocation", func(t *testing.T) {
		resp, err := c.GetTargetBalanceAllocation(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)

		assert.Contains(t,
			[]RestingMarginReservation{RestingMarginReservationMax, RestingMarginReservationSum},
			resp.RestingMarginReservation)

		if len(resp.Allocations) == 0 {
			t.Log("automatic rebalancing is disabled (no allocations)")
			return
		}
		total := 0
		for _, a := range resp.Allocations {
			total += a.Percent
		}
		assert.Equal(t, 100, total, "allocation percentages must total 100")
	})
}

func TestHTTPIntegration_IntraExchangeTransfersRead(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetIntraExchangeInstanceTransfers", func(t *testing.T) {
		resp, err := c.GetIntraExchangeInstanceTransfers(ctx,
			GetIntraExchangeInstanceTransfersParams{Limit: 10})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotNil(t, resp.Transfers)
		assert.LessOrEqual(t, len(resp.Transfers), 10, "limit must be honored")
		t.Logf("transfers=%d", len(resp.Transfers))

		for _, tr := range resp.Transfers {
			assert.NotEmpty(t, tr.TransferID)
			assert.Contains(t,
				[]IntraExchangeInstanceTransferStatus{
					IntraExchangeInstanceTransferStatusPending,
					IntraExchangeInstanceTransferStatusComplete,
				}, tr.Status)
		}
	})

	t.Run("GetIntraExchangeInstanceTransfer", func(t *testing.T) {
		_, err := c.GetIntraExchangeInstanceTransfer(ctx, "00000000-0000-0000-0000-000000000000")
		require.Error(t, err)
		assert.True(t, isAPIErrorCode(err, 404),
			"an unknown transfer id should be a structured 404, got: %v", err)
	})
}

func TestHTTPIntegration_BlockTradeProposalsRead(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetBlockTradeProposals", func(t *testing.T) {
		resp, err := c.GetBlockTradeProposals(ctx, GetBlockTradeProposalsParams{Limit: 10})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotNil(t, resp.BlockTradeProposals)
		t.Logf("block trade proposals=%d", len(resp.BlockTradeProposals))

		for _, p := range resp.BlockTradeProposals {
			assert.NotEmpty(t, p.ID)
			assert.Contains(t, []string{"yes", "no"}, p.MakerSide)
		}
	})

	t.Run("AcceptBlockTradeProposal", func(t *testing.T) {
		// Accepting executes a real trade and cannot be undone, so the only
		// safe live assertion is that an unknown proposal is rejected cleanly.
		err := c.AcceptBlockTradeProposal(ctx, "00000000-0000-0000-0000-000000000000",
			AcceptBlockTradeProposalRequest{})
		require.Error(t, err)
		assert.True(t, isAPIErrorCode(err, 404) || isAPIErrorCode(err, 400) || isAPIErrorCode(err, 403),
			"unknown proposal should be a structured 4xx, got: %v", err)
	})
}

// The RFQ-scoped quote endpoints need a counterparty: a quote carries distinct
// creator_id and rfq_creator_id, and a single key cannot quote its own RFQ.
// Accepting or confirming executes a real trade with no way back. What is worth
// asserting live, and is asserted here, is the error contract — path
// construction, request signing and error decoding all exercised, with no state
// created and nothing to clean up.
func TestHTTPIntegration_RFQScopedQuotes(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	const (
		unknownRFQ   = "00000000-0000-0000-0000-000000000000"
		unknownQuote = "11111111-1111-1111-1111-111111111111"
	)

	assertUnknown := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		assert.True(t, isAPIErrorCode(err, 404) || isAPIErrorCode(err, 400),
			"unknown rfq/quote should be a structured 4xx rather than a parse failure, got: %v", err)
	}

	t.Run("GetRFQQuote", func(t *testing.T) {
		_, err := c.GetRFQQuote(ctx, unknownRFQ, unknownQuote)
		assertUnknown(t, err)
	})

	t.Run("DeleteRFQQuote", func(t *testing.T) {
		assertUnknown(t, c.DeleteRFQQuote(ctx, unknownRFQ, unknownQuote))
	})

	t.Run("AcceptRFQQuote", func(t *testing.T) {
		assertUnknown(t, c.AcceptRFQQuote(ctx, unknownRFQ, unknownQuote,
			AcceptQuoteRequest{AcceptedSide: SideYes}))
	})

	t.Run("ConfirmRFQQuote", func(t *testing.T) {
		assertUnknown(t, c.ConfirmRFQQuote(ctx, unknownRFQ, unknownQuote))
	})
}

// FCM endpoints require Futures Commission Merchant membership. Without it the
// contract fcm.go promises is that the caller sees a structured *APIError
// rather than a parse failure — that is exactly what is asserted.
func TestHTTPIntegration_FCM(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	assertGated := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Skip("account has FCM access — nothing to assert about the gate")
		}
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "FCM gating must surface as *APIError, not a parse failure")
		assert.Contains(t, []int{400, 401, 403, 404}, apiErr.StatusCode)
		t.Logf("FCM gated with %d: %s", apiErr.StatusCode, apiErr.Code)
	}

	t.Run("GetFCMOrders", func(t *testing.T) {
		_, err := c.GetFCMOrders(ctx, GetFCMOrdersParams{SubtraderID: "nonexistent_1"})
		assertGated(t, err)
	})

	t.Run("GetFCMPositions", func(t *testing.T) {
		_, err := c.GetFCMPositions(ctx, GetFCMPositionsParams{SubtraderID: "nonexistent_1"})
		assertGated(t, err)
	})
}
