package gokalshi

import (
	"context"
	"fmt"
)

const pathStructuredTargets = "/trade-api/v2/structured_targets"

// GetStructuredTargets — Get Structured Targets
//
// GET /trade-api/v2/structured_targets
//
// Page size (min: 1, max: 2000)
//
// See https://trading-api.readme.io/reference/getstructuredtargets
func (c *Client) GetStructuredTargets(ctx context.Context, params GetStructuredTargetsParams) (GetStructuredTargetsResponse, error) {
	return getJSON[GetStructuredTargetsResponse](c, ctx, pathStructuredTargets, params.toMap())
}

// GetStructuredTarget — Get Structured Target
//
// GET /trade-api/v2/structured_targets/{structured_target_id}
//
// Endpoint for getting data about a specific structured target by its ID.
//
// See https://trading-api.readme.io/reference/getstructuredtarget
func (c *Client) GetStructuredTarget(ctx context.Context, structuredTargetID string) (GetStructuredTargetResponse, error) {
	path := fmt.Sprintf("%s/%s", pathStructuredTargets, structuredTargetID)
	return getJSON[GetStructuredTargetResponse](c, ctx, path, nil)
}

// GetStructuredTargetsParams are query parameters for GetStructuredTargets.
type GetStructuredTargetsParams struct {
	Cursor      string
	Limit       int
	EventTicker string
}

func (p GetStructuredTargetsParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		String("event_ticker", p.EventTicker).
		Build()
}
