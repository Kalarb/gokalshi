package gokalshi

import (
	"context"
	"fmt"
)

// GetSeries — Get Series
//
// GET /trade-api/v2/series/{series_ticker}
//
// Endpoint for getting data about a specific series by its ticker. A series
// represents a template for recurring events that follow the same format and
// rules (e.g., "Monthly Jobs Report", "Weekly Initial Jobless Claims", "Daily
// Weather in NYC"). Series define the structure, settlement sources, and
// metadata that will be applied to each recurring event instance within that
// series.
//
// See https://trading-api.readme.io/reference/getseries
func (c *Client) GetSeries(ctx context.Context, seriesTicker string, params GetSeriesParams) (GetSeriesResponse, error) {
	path := fmt.Sprintf("%s/%s", pathSeries, seriesTicker)
	return getJSON[GetSeriesResponse](c, ctx, path, params.toMap())
}

// GetSeriesList — Get Series List
//
// GET /trade-api/v2/series
//
// Endpoint for getting data about multiple series with specified filters. A
// series represents a template for recurring events that follow the same
// format and rules (e.g., "Monthly Jobs Report", "Weekly Initial Jobless
// Claims", "Daily Weather in NYC"). This endpoint allows you to browse and
// discover available series templates by category.
//
// See https://trading-api.readme.io/reference/getserieslist
func (c *Client) GetSeriesList(ctx context.Context, params GetSeriesListParams) (GetSeriesListResponse, error) {
	return getJSON[GetSeriesListResponse](c, ctx, pathSeries, params.toMap())
}

// ---------------------------------------------------------------------------
// Query parameter types
// ---------------------------------------------------------------------------

// GetSeriesParams holds optional query parameters for GetSeries.
type GetSeriesParams struct {
	IncludeVolume bool
}

func (p GetSeriesParams) toMap() map[string]string {
	return NewQuery().
		Bool("include_volume", p.IncludeVolume).
		Build()
}

// GetSeriesListParams holds optional query parameters for GetSeriesList.
type GetSeriesListParams struct {
	Category               string
	Tags                   string
	IncludeProductMetadata bool
	IncludeVolume          bool
	MinUpdatedTs           int64
}

func (p GetSeriesListParams) toMap() map[string]string {
	return NewQuery().
		String("category", p.Category).
		String("tags", p.Tags).
		Bool("include_product_metadata", p.IncludeProductMetadata).
		Bool("include_volume", p.IncludeVolume).
		Int64("min_updated_ts", p.MinUpdatedTs).
		Build()
}
