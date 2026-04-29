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

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// SettlementSource is a source used for market determination.
type SettlementSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// SeriesDetail is the full series object returned by the Kalshi API.
type SeriesDetail struct {
	Ticker                 string             `json:"ticker"`
	Frequency              string             `json:"frequency"`
	Title                  string             `json:"title"`
	Category               string             `json:"category"`
	Tags                   []string           `json:"tags"`
	SettlementSources      []SettlementSource `json:"settlement_sources"`
	ContractURL            string             `json:"contract_url"`
	ContractTermsURL       string             `json:"contract_terms_url"`
	FeeType                FeeType            `json:"fee_type"`
	FeeMultiplier          float64            `json:"fee_multiplier"`
	AdditionalProhibitions []string           `json:"additional_prohibitions"`
	ProductMetadata        any                `json:"product_metadata"`
	VolumeFP               string             `json:"volume_fp"`
	LastUpdatedTs          string             `json:"last_updated_ts"`
}

// GetSeriesResponse is the response from GET /series/{ticker}.
type GetSeriesResponse struct {
	Series SeriesDetail `json:"series"`
}

// GetSeriesListResponse is the response from GET /series.
type GetSeriesListResponse struct {
	Series []SeriesDetail `json:"series"`
}
