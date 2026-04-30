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
func (c *Client) GetTagsByCategories(ctx context.Context) (GetTagsForSeriesCategoriesResponse, error) {
	return getJSON[GetTagsForSeriesCategoriesResponse](c, ctx, pathSearch+"/tags_by_categories", nil)
}

// GetFiltersBySport — Get Filters for Sports
//
// GET /trade-api/v2/search/filters_by_sport
//
// Retrieve available filters organized by sport.
//
// See https://trading-api.readme.io/reference/getfiltersforsports
func (c *Client) GetFiltersBySport(ctx context.Context) (GetFiltersBySportsResponse, error) {
	return getJSON[GetFiltersBySportsResponse](c, ctx, pathSearch+"/filters_by_sport", nil)
}
