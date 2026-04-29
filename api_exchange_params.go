package gokalshi

// Query parameter structs for exchange-related API endpoints.

// GetSeriesFeeChangesParams holds optional query parameters for GetSeriesFeeChanges.
type GetSeriesFeeChangesParams struct {
	SeriesTicker   string
	ShowHistorical bool
}

func (p GetSeriesFeeChangesParams) toMap() map[string]string {
	return NewQuery().
		String("series_ticker", p.SeriesTicker).
		Bool("show_historical", p.ShowHistorical).
		Build()
}
