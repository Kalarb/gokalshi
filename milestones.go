package gokalshi

import (
	"context"
	"fmt"
)

const pathMilestones = "/trade-api/v2/milestones"

// GetMilestones — Get Milestones
//
// GET /trade-api/v2/milestones
//
// Minimum start date to filter milestones. Format: RFC3339 timestamp
//
// See https://trading-api.readme.io/reference/getmilestones
func (c *Client) GetMilestones(ctx context.Context, params GetMilestonesParams) (GetMilestonesResponse, error) {
	return getJSON[GetMilestonesResponse](c, ctx, pathMilestones, params.toMap())
}

// GetMilestone — Get Milestone
//
// GET /trade-api/v2/milestones/{milestone_id}
//
// Endpoint for getting data about a specific milestone by its ID.
//
// See https://trading-api.readme.io/reference/getmilestone
func (c *Client) GetMilestone(ctx context.Context, milestoneID string) (GetMilestoneResponse, error) {
	path := fmt.Sprintf("%s/%s", pathMilestones, milestoneID)
	return getJSON[GetMilestoneResponse](c, ctx, path, nil)
}

// GetMilestonesParams are query parameters for GetMilestones.
type GetMilestonesParams struct {
	Cursor      string
	Limit       int
	EventTicker string
}

func (p GetMilestonesParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		Int("limit", p.Limit).
		String("event_ticker", p.EventTicker).
		Build()
}
