package gokalshi

import (
	"context"
)

const pathSearch = "/trade-api/v2/search"

// GetTagsByCategories — Get Tags for Series Categories
//
// GET /trade-api/v2/search/tags_by_categories
//
// Retrieve tags organized by series categories.
//
// See https://trading-api.readme.io/reference/gettagsforseriescategories
func (c *Client) GetTagsByCategories(ctx context.Context) (GetTagsByCategoriesResponse, error) {
	return getJSON[GetTagsByCategoriesResponse](c, ctx, pathSearch+"/tags_by_categories", nil)
}

// GetFiltersBySport — Get Filters for Sports
//
// GET /trade-api/v2/search/filters_by_sport
//
// Retrieve available filters organized by sport.
//
// See https://trading-api.readme.io/reference/getfiltersforsports
func (c *Client) GetFiltersBySport(ctx context.Context) (GetFiltersBySportResponse, error) {
	return getJSON[GetFiltersBySportResponse](c, ctx, pathSearch+"/filters_by_sport", nil)
}

// GetTagsByCategoriesResponse is the response from GET /search/tags_by_categories.
type GetTagsByCategoriesResponse struct {
	TagsByCategories map[string][]string `json:"tags_by_categories"`
}

// SportFilters holds filter details for a single sport.
type SportFilters struct {
	Scopes       []string                         `json:"scopes"`
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
