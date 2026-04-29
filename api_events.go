package gokalshi

import (
	"context"
	"fmt"
)

const pathEvents = "/trade-api/v2/events"

// GetEvent retrieves details for a single event.
func (c *Client) GetEvent(ctx context.Context, eventTicker string, params GetEventParams) (GetEventResponse, error) {
	path := fmt.Sprintf("%s/%s", pathEvents, eventTicker)
	return getJSON[GetEventResponse](c, ctx, path, params.toMap())
}

// GetEvents retrieves events matching the given parameters.
func (c *Client) GetEvents(ctx context.Context, params GetEventsParams) (GetEventsResponse, error) {
	return getJSON[GetEventsResponse](c, ctx, pathEvents, params.toMap())
}

// GetEventMetadata retrieves metadata for a single event.
func (c *Client) GetEventMetadata(ctx context.Context, eventTicker string) (GetEventMetadataResponse, error) {
	path := fmt.Sprintf("%s/%s/metadata", pathEvents, eventTicker)
	return getJSON[GetEventMetadataResponse](c, ctx, path, nil)
}

// GetMultivariateEvents retrieves multivariate (combo) events.
func (c *Client) GetMultivariateEvents(ctx context.Context, params GetMultivariateEventsParams) (GetMultivariateEventsResponse, error) {
	return getJSON[GetMultivariateEventsResponse](c, ctx, pathEvents+"/multivariate", params.toMap())
}

// GetEventCandlesticks retrieves aggregated candlestick data for all markets in an event.
func (c *Client) GetEventCandlesticks(ctx context.Context, seriesTicker, eventTicker string, params GetEventCandlesticksParams) (GetEventCandlesticksResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/events/%s/candlesticks", seriesTicker, eventTicker)
	return getJSON[GetEventCandlesticksResponse](c, ctx, path, params.toMap())
}

// GetEventForecastPercentileHistory retrieves historical forecast percentile data for an event.
func (c *Client) GetEventForecastPercentileHistory(ctx context.Context, seriesTicker, eventTicker string, params GetEventForecastPercentileHistoryParams) (GetEventForecastPercentileHistoryResponse, error) {
	path := fmt.Sprintf("/trade-api/v2/series/%s/events/%s/forecast_percentile_history", seriesTicker, eventTicker)
	return getJSON[GetEventForecastPercentileHistoryResponse](c, ctx, path, params.toMap())
}
