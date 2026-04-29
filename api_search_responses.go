package gokalshi

// Response types for search-related API endpoints.

// GetTagsByCategoriesResponse is the response from GET /search/tags_by_categories.
type GetTagsByCategoriesResponse struct {
	TagsByCategories map[string][]string `json:"tags_by_categories"`
}

// SportFilters holds filter details for a single sport.
type SportFilters struct {
	Scopes       []string                        `json:"scopes"`
	Competitions map[string]SportCompetitionScope `json:"competitions"`
}

// SportCompetitionScope holds scopes for a competition within a sport.
type SportCompetitionScope struct {
	Scopes []string `json:"scopes"`
}

// GetFiltersBySportResponse is the response from GET /search/filters_by_sport.
type GetFiltersBySportResponse struct {
	FiltersBySports map[string]SportFilters `json:"filters_by_sports"`
	SportOrdering   []string                `json:"sport_ordering"`
}
