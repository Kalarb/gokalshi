package gokalshi

// Query parameter structs for series-related API endpoints.

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
