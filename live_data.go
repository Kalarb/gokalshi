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
// See https://docs.kalshi.com/api-reference/live-data/get-live-data-by-milestone
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
// See https://docs.kalshi.com/api-reference/live-data/get-live-data
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
// Get live data for an event by its event ticker. Serves event-keyed live data
// such as crypto price charts, commodity price timeseries, and weather
// observations. The `type` field in the response names the schema of the
// `details` object.
//
// See https://docs.kalshi.com/api-reference/live-data/get-event-live-data
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
// Get the Kalshi-computed city temperature index: the canonical
// minute-resolution series behind hourly temperature markets. City-keyed and
// independent of any event — the series exists whenever the city's index is
// configured. Values are Fahrenheit rounded to 0.01. Minutes where the index
// quorum failed carry no value and are never returned as points, so gaps in
// the series are real gaps. With `detailed=true` each point additionally
// carries every member station's reported reading and quality-control
// disposition — the pre-incorporation breakdown.
//
// See https://docs.kalshi.com/api-reference/live-data/get-weather-index
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
// Get a city's published weather-index configuration timeline: the launch
// configuration plus every weekly offset calibration and methodology update,
// ascending by effective time. Each record carries the station weights,
// station offsets (Celsius), and the city reference used to compute index
// values from the record's effective time until the next record — everything
// needed to reproduce published index values for minutes computed under that
// configuration version. Offsets are re-estimated weekly after each complete
// UTC week; weights never change through calibration. The timeline is
// append-only and complete: it is never trimmed.
//
// See https://docs.kalshi.com/api-reference/live-data/get-weather-index-calibrations
func (c *Client) GetWeatherIndexCalibrations(ctx context.Context, city string) (GetWeatherIndexCalibrationsResponse, error) {
	path := fmt.Sprintf("%s/weather/%s/calibrations", pathLiveData, city)
	return getJSON[GetWeatherIndexCalibrationsResponse](c, ctx, path, nil)
}
