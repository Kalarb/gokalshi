package gokalshi

import (
	"context"
	"fmt"
)

// GetMultivariateEventCollections — Get Multivariate Event Collections
//
// GET /trade-api/v2/multivariate_event_collections
//
// Endpoint for getting data about multivariate event collections.
//
// See https://docs.kalshi.com/api-reference/multivariate/get-multivariate-event-collections
func (c *Client) GetMultivariateEventCollections(ctx context.Context, params GetMultivariateEventCollectionsParams) (GetMultivariateEventCollectionsResponse, error) {
	return getJSON[GetMultivariateEventCollectionsResponse](c, ctx, pathMVECollections, params.toMap())
}

// GetMultivariateEventCollection — Get Multivariate Event Collection
//
// GET /trade-api/v2/multivariate_event_collections/{collection_ticker}
//
// Endpoint for getting data about a multivariate event collection by its
// ticker.
//
// See https://docs.kalshi.com/api-reference/multivariate/get-multivariate-event-collection
func (c *Client) GetMultivariateEventCollection(ctx context.Context, collectionTicker string) (GetMultivariateEventCollectionResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMVECollections, collectionTicker)
	return getJSON[GetMultivariateEventCollectionResponse](c, ctx, path, nil)
}

// CreateMarketInMultivariateEventCollection — Create Market In Multivariate Event Collection
//
// POST /trade-api/v2/multivariate_event_collections/{collection_ticker}
//
// Endpoint for creating an individual market in a multivariate event
// collection. This endpoint must be hit at least once before trading or
// looking up a market. Users are limited to 5000 creations per week.
//
// See https://docs.kalshi.com/api-reference/multivariate/create-market-in-multivariate-event-collection
func (c *Client) CreateMarketInMultivariateEventCollection(ctx context.Context, collectionTicker string, req CreateMarketInMultivariateEventCollectionRequest) (CreateMarketInMultivariateEventCollectionResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMVECollections, collectionTicker)
	return postJSON[CreateMarketInMultivariateEventCollectionResponse](c, ctx, path, req, 10.0)
}

// GetMultivariateEventCollectionsParams are query parameters for GetMultivariateEventCollections.
type GetMultivariateEventCollectionsParams struct {
	Cursor string
	Limit  int
}

func (p GetMultivariateEventCollectionsParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		Build()
}

// GetMVECollectionLookupParams are query parameters for GetMultivariateEventCollectionLookupHistory.
type GetMVECollectionLookupParams struct {
	Cursor string
	Limit  int
}

func (p GetMVECollectionLookupParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		Build()
}
