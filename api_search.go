package gokalshi

import (
	"context"
)

const pathSearch = "/trade-api/v2/search"

// GetTagsByCategories retrieves tags organized by series categories.
func (c *Client) GetTagsByCategories(ctx context.Context) (GetTagsByCategoriesResponse, error) {
	return getJSON[GetTagsByCategoriesResponse](c, ctx, pathSearch+"/tags_by_categories", nil)
}

// GetFiltersBySport retrieves available filters organized by sport.
func (c *Client) GetFiltersBySport(ctx context.Context) (GetFiltersBySportResponse, error) {
	return getJSON[GetFiltersBySportResponse](c, ctx, pathSearch+"/filters_by_sport", nil)
}
