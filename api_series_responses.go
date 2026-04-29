package gokalshi

// Response types for series-related API endpoints.

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
