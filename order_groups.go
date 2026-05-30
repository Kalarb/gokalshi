package gokalshi

import (
	"context"
	"fmt"
)

// CreateOrderGroup — Create Order Group
//
// POST /trade-api/v2/portfolio/order_groups/create
//
// Creates a new order group with a contracts limit measured over a rolling
// 15-second window. When the limit is hit, all orders in the group are
// cancelled and no new orders can be placed until reset.
//
// See https://trading-api.readme.io/reference/createordergroup
func (c *Client) CreateOrderGroup(ctx context.Context, req CreateOrderGroupRequest) (CreateOrderGroupResponse, error) {
	return postJSON[CreateOrderGroupResponse](c, ctx, pathOrderGroups+"/create", req, 10.0)
}

// GetOrderGroups — Get Order Groups
//
// GET /trade-api/v2/portfolio/order_groups
//
// Retrieves all order groups for the authenticated user.
//
// See https://trading-api.readme.io/reference/getordergroups
func (c *Client) GetOrderGroups(ctx context.Context, params GetOrderGroupsParams) (GetOrderGroupsResponse, error) {
	return getJSON[GetOrderGroupsResponse](c, ctx, pathOrderGroups, params.toMap())
}

// GetOrderGroup — Get Order Group
//
// GET /trade-api/v2/portfolio/order_groups/{order_group_id}
//
// Retrieves details for a single order group including all order IDs and
// auto-cancel status.
//
// See https://trading-api.readme.io/reference/getordergroup
func (c *Client) GetOrderGroup(ctx context.Context, orderGroupID string, params GetOrderGroupParams) (GetOrderGroupResponse, error) {
	path := fmt.Sprintf("%s/%s", pathOrderGroups, orderGroupID)
	return getJSON[GetOrderGroupResponse](c, ctx, path, params.toMap())
}

// DeleteOrderGroup — Delete Order Group
//
// DELETE /trade-api/v2/portfolio/order_groups/{order_group_id}
//
// Deletes an order group and cancels all orders within it. This permanently
// removes the group.
//
// See https://trading-api.readme.io/reference/deleteordergroup
func (c *Client) DeleteOrderGroup(ctx context.Context, orderGroupID string, params DeleteOrderGroupParams) error {
	path := fmt.Sprintf("%s/%s", pathOrderGroups, orderGroupID)
	_, err := c.do(ctx, "DELETE", path, 0, 10.0, nil, params.toMap())
	return err
}

// ResetOrderGroup — Reset Order Group
//
// PUT /trade-api/v2/portfolio/order_groups/{order_group_id}/reset
//
// Resets the order group's matched contracts counter to zero, allowing new
// orders to be placed again after the limit was hit.
//
// See https://trading-api.readme.io/reference/resetordergroup
func (c *Client) ResetOrderGroup(ctx context.Context, orderGroupID string, params OrderGroupActionParams) error {
	path := fmt.Sprintf("%s/%s/reset", pathOrderGroups, orderGroupID)
	_, err := c.do(ctx, "PUT", path, 0, 10.0, nil, params.toMap())
	return err
}

// TriggerOrderGroup — Trigger Order Group
//
// PUT /trade-api/v2/portfolio/order_groups/{order_group_id}/trigger
//
// Triggers the order group, canceling all orders in the group and preventing
// new orders until the group is reset.
//
// See https://trading-api.readme.io/reference/triggerordergroup
func (c *Client) TriggerOrderGroup(ctx context.Context, orderGroupID string, params OrderGroupActionParams) error {
	path := fmt.Sprintf("%s/%s/trigger", pathOrderGroups, orderGroupID)
	_, err := c.do(ctx, "PUT", path, 0, 10.0, nil, params.toMap())
	return err
}

// UpdateOrderGroupLimit — Update Order Group Limit
//
// PUT /trade-api/v2/portfolio/order_groups/{order_group_id}/limit
//
// Updates the order group contracts limit (rolling 15-second window). If the
// updated limit would immediately trigger the group, all orders in the group
// are canceled and the group is triggered.
//
// See https://trading-api.readme.io/reference/updateordergrouplimit
func (c *Client) UpdateOrderGroupLimit(ctx context.Context, orderGroupID string, req UpdateOrderGroupLimitRequest, params UpdateOrderGroupLimitParams) error {
	path := fmt.Sprintf("%s/%s/limit", pathOrderGroups, orderGroupID)
	_, err := c.do(ctx, "PUT", path, 0, 10.0, req, params.toMap())
	return err
}

// GetOrderGroupsParams are query parameters for GetOrderGroups.
type GetOrderGroupsParams struct {
	Subaccount string
}

func (p GetOrderGroupsParams) toMap() map[string]string {
	return NewQuery().
		String("subaccount", p.Subaccount).
		Build()
}

// GetOrderGroupParams are query parameters for GetOrderGroup.
type GetOrderGroupParams struct {
	Subaccount string
}

func (p GetOrderGroupParams) toMap() map[string]string {
	return NewQuery().
		String("subaccount", p.Subaccount).
		Build()
}

// DeleteOrderGroupParams are query parameters for DeleteOrderGroup.
type DeleteOrderGroupParams struct {
	Subaccount    string
	ExchangeIndex int
}

func (p DeleteOrderGroupParams) toMap() map[string]string {
	return NewQuery().
		String("subaccount", p.Subaccount).
		Int("exchange_index", p.ExchangeIndex).
		Build()
}

// OrderGroupActionParams are query parameters for ResetOrderGroup and TriggerOrderGroup.
type OrderGroupActionParams struct {
	Subaccount    string
	ExchangeIndex int
}

func (p OrderGroupActionParams) toMap() map[string]string {
	return NewQuery().
		String("subaccount", p.Subaccount).
		Int("exchange_index", p.ExchangeIndex).
		Build()
}

// UpdateOrderGroupLimitParams are query parameters for UpdateOrderGroupLimit.
type UpdateOrderGroupLimitParams struct {
	ExchangeIndex int
}

func (p UpdateOrderGroupLimitParams) toMap() map[string]string {
	return NewQuery().
		Int("exchange_index", p.ExchangeIndex).
		Build()
}
