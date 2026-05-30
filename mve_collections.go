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
// See https://trading-api.readme.io/reference/getmultivariateeventcollections
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
// See https://trading-api.readme.io/reference/getmultivariateeventcollection
func (c *Client) GetMultivariateEventCollection(ctx context.Context, collectionTicker string) (GetMultivariateEventCollectionResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMVECollections, collectionTicker)
	return getJSON[GetMultivariateEventCollectionResponse](c, ctx, path, nil)
}

// GetMultivariateEventCollectionLookupHistory — Get Multivariate Event Collection Lookup History
//
// GET /trade-api/v2/multivariate_event_collections/{collection_ticker}/lookup
//
// DEPRECATED: This endpoint predates RFQs and should not be used for new
// integrations. Endpoint for retrieving which markets in an event collection
// were recently looked up.
//
// See https://trading-api.readme.io/reference/getmultivariateeventcollectionlookuphistory
func (c *Client) GetMultivariateEventCollectionLookupHistory(ctx context.Context, collectionTicker string, params GetMVECollectionLookupParams) (GetMultivariateEventCollectionLookupHistoryResponse, error) {
	path := fmt.Sprintf("%s/%s/lookup", pathMVECollections, collectionTicker)
	return getJSON[GetMultivariateEventCollectionLookupHistoryResponse](c, ctx, path, params.toMap())
}

// CreateMarketInMultivariateEventCollection — Create Market In Multivariate Event Collection
//
// POST /trade-api/v2/multivariate_event_collections/{collection_ticker}
//
// Endpoint for creating an individual market in a multivariate event
// collection. This endpoint must be hit at least once before trading or
// looking up a market. Users are limited to 5000 creations per week.
//
// See https://trading-api.readme.io/reference/createmarketinmultivariateeventcollection
func (c *Client) CreateMarketInMultivariateEventCollection(ctx context.Context, collectionTicker string, req CreateMarketInMultivariateEventCollectionRequest) (CreateMarketInMultivariateEventCollectionResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMVECollections, collectionTicker)
	return postJSON[CreateMarketInMultivariateEventCollectionResponse](c, ctx, path, req, 10.0)
}

// LookupTickersForMarketInMultivariateEventCollection — Lookup Tickers For Market In Multivariate Event Collection
//
// PUT /trade-api/v2/multivariate_event_collections/{collection_ticker}/lookup
//
// DEPRECATED: This endpoint predates RFQs and should not be used for new
// integrations. Endpoint for looking up an individual market in a multivariate
// event collection. If CreateMarketInMultivariateEventCollection has never
// been hit with that variable combination before, this will return a 404.
//
// See https://trading-api.readme.io/reference/lookuptickersformarketinmultivariateeventcollection
func (c *Client) LookupTickersForMarketInMultivariateEventCollection(ctx context.Context, collectionTicker string, req LookupTickersForMarketInMultivariateEventCollectionRequest) (LookupTickersForMarketInMultivariateEventCollectionResponse, error) {
	path := fmt.Sprintf("%s/%s/lookup", pathMVECollections, collectionTicker)
	return putJSON[LookupTickersForMarketInMultivariateEventCollectionResponse](c, ctx, path, req, 10.0)
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
