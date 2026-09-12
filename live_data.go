package gokalshi

import (
	"context"
	"fmt"
)

// GetLiveDataBatch — Get Live Data Batch
//
// GET /trade-api/v2/live_data/batch
func (c *Client) GetLiveDataBatch(ctx context.Context, params GetLiveDataBatchParams) (GetLiveDatasResponse, error) {
	return getJSON[GetLiveDatasResponse](c, ctx, pathLiveData+"/batch", params.toMap())
}

// GetLiveDataByMilestone — Get Live Data
//
// GET /trade-api/v2/live_data/milestone/{milestone_id}
//
// Get live data for a specific milestone.
//
// See https://trading-api.readme.io/reference/getlivedatabymilestone
func (c *Client) GetLiveDataByMilestone(ctx context.Context, milestoneID string, params GetLiveDataParams) (GetLiveDataResponse, error) {
	path := fmt.Sprintf("%s/milestone/%s", pathLiveData, milestoneID)
	return getJSON[GetLiveDataResponse](c, ctx, path, params.toMap())
}

// GetMilestoneGameStats — Get Game Stats
//
// GET /trade-api/v2/live_data/milestone/{milestone_id}/game_stats
func (c *Client) GetMilestoneGameStats(ctx context.Context, milestoneID string) (GetGameStatsResponse, error) {
	path := fmt.Sprintf("%s/milestone/%s/game_stats", pathLiveData, milestoneID)
	return getJSON[GetGameStatsResponse](c, ctx, path, nil)
}

// GetLiveData — Get Live Data (with type)
//
// GET /trade-api/v2/live_data/{type}/milestone/{milestone_id}
//
// Get live data for a specific milestone. This is the legacy endpoint that
// requires a type path parameter. Prefer using
// `/live_data/milestone/{milestone_id}` instead.
//
// See https://trading-api.readme.io/reference/getlivedata
func (c *Client) GetLiveData(ctx context.Context, dataType, milestoneID string, params GetLiveDataParams) (GetLiveDataResponse, error) {
	path := fmt.Sprintf("%s/%s/milestone/%s", pathLiveData, dataType, milestoneID)
	return getJSON[GetLiveDataResponse](c, ctx, path, params.toMap())
}

// GetLiveDataBatchParams are query parameters for GetLiveDataBatch.
type GetLiveDataBatchParams struct {
	MilestoneIDs string
}

func (p GetLiveDataBatchParams) toMap() map[string]string {
	return NewQuery().
		String("milestone_ids", p.MilestoneIDs).
		Build()
}

// GetLiveDataParams are query parameters for GetLiveDataByMilestone and GetLiveData.
type GetLiveDataParams struct {
	IncludePlayerStats bool
}

func (p GetLiveDataParams) toMap() map[string]string {
	return NewQuery().
		Bool("include_player_stats", p.IncludePlayerStats).
		Build()
}

// GetEventLiveData — Get Event Live Data
//
// GET /trade-api/v2/live_data/events/{event_ticker}
//
// Live data keyed by event rather than by milestone, which saves resolving the
// milestone id first when all you have is the event.
func (c *Client) GetEventLiveData(ctx context.Context, eventTicker string, params GetEventLiveDataParams) (GetEventLiveDataResponse, error) {
	path := fmt.Sprintf("%s/events/%s", pathLiveData, eventTicker)
	return getJSON[GetEventLiveDataResponse](c, ctx, path, params.toMap())
}

// GetEventLiveDataParams are the query parameters for GetEventLiveData.
type GetEventLiveDataParams struct {
	Range string
}

func (p GetEventLiveDataParams) toMap() map[string]string {
	return NewQuery().String("range", p.Range).Build()
}

// GetWeatherIndex — Get Weather Index
//
// GET /trade-api/v2/live_data/weather/{city}
//
// Index readings underlying that city's weather markets.
func (c *Client) GetWeatherIndex(ctx context.Context, city string, params GetWeatherIndexParams) (GetWeatherIndexResponse, error) {
	path := fmt.Sprintf("%s/weather/%s", pathLiveData, city)
	return getJSON[GetWeatherIndexResponse](c, ctx, path, params.toMap())
}

// GetWeatherIndexParams are the query parameters for GetWeatherIndex.
type GetWeatherIndexParams struct {
	From     int64
	To       int64
	LastSec  int
	Detailed bool
}

func (p GetWeatherIndexParams) toMap() map[string]string {
	return NewQuery().
		Int64("from", p.From).
		Int64("to", p.To).
		Int("last_sec", p.LastSec).
		Bool("detailed", p.Detailed).
		Build()
}

// GetWeatherIndexCalibrations — Get Weather Index Calibrations
//
// GET /trade-api/v2/live_data/weather/{city}/calibrations
//
// The station calibration history behind a city's weather index.
func (c *Client) GetWeatherIndexCalibrations(ctx context.Context, city string) (GetWeatherIndexCalibrationsResponse, error) {
	path := fmt.Sprintf("%s/weather/%s/calibrations", pathLiveData, city)
	return getJSON[GetWeatherIndexCalibrationsResponse](c, ctx, path, nil)
}
