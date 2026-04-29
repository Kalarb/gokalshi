package gokalshi

import (
	"context"
	"fmt"
)

const pathSeries = "/trade-api/v2/series"

// GetSeries retrieves details for a single series.
func (c *Client) GetSeries(ctx context.Context, seriesTicker string, params GetSeriesParams) (GetSeriesResponse, error) {
	path := fmt.Sprintf("%s/%s", pathSeries, seriesTicker)
	return getJSON[GetSeriesResponse](c, ctx, path, params.toMap())
}

// GetSeriesList retrieves a list of series.
func (c *Client) GetSeriesList(ctx context.Context, params GetSeriesListParams) (GetSeriesListResponse, error) {
	return getJSON[GetSeriesListResponse](c, ctx, pathSeries, params.toMap())
}
