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

	t.Run("GetMultivariateEventCollectionLookupHistory", func(t *testing.T) {
		_, err := c.GetMultivariateEventCollectionLookupHistory(ctx, collectionTicker, GetMVECollectionLookupParams{Limit: 5})
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
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
