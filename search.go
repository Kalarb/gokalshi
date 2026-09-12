package gokalshi

import (
	"context"
)

// GetTagsByCategories — Get Tags for Series Categories
//
// GET /trade-api/v2/search/tags_by_categories
//
// Retrieve tags organized by series categories.
//
// See https://docs.kalshi.com/api-reference/search/get-tags-for-series-categories
func (c *Client) GetTagsByCategories(ctx context.Context) (GetTagsForSeriesCategoriesResponse, error) {
	return getJSON[GetTagsForSeriesCategoriesResponse](c, ctx, pathSearch+"/tags_by_categories", nil)
}

// GetFiltersBySport — Get Filters for Sports
//
// GET /trade-api/v2/search/filters_by_sport
//
// Retrieve available filters organized by sport.
//
// See https://docs.kalshi.com/api-reference/search/get-filters-for-sports
func (c *Client) GetFiltersBySport(ctx context.Context) (GetFiltersBySportsResponse, error) {
	return getJSON[GetFiltersBySportsResponse](c, ctx, pathSearch+"/filters_by_sport", nil)
}
