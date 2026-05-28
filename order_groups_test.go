package gokalshi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrderGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/create", r.URL.Path)
		fmt.Fprint(w, `{"order_group_id":"og-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	resp, err := c.CreateOrderGroup(context.Background(), CreateOrderGroupRequest{})
	require.NoError(t, err)
	assert.Equal(t, "og-1", resp.OrderGroupID)
}

func TestGetOrderGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups", r.URL.Path)
		fmt.Fprint(w, `{"order_groups":[]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetOrderGroups(context.Background(), GetOrderGroupsParams{})
	require.NoError(t, err)
}

func TestGetOrderGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/og-1", r.URL.Path)
		fmt.Fprint(w, `{"order_group":{}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetOrderGroup(context.Background(), "og-1", GetOrderGroupParams{})
	require.NoError(t, err)
}

func TestDeleteOrderGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/og-1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteOrderGroup(context.Background(), "og-1", DeleteOrderGroupParams{})
	require.NoError(t, err)
}

func TestResetOrderGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/og-1/reset", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.ResetOrderGroup(context.Background(), "og-1", OrderGroupActionParams{})
	require.NoError(t, err)
}

func TestTriggerOrderGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/og-1/trigger", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.TriggerOrderGroup(context.Background(), "og-1", OrderGroupActionParams{})
	require.NoError(t, err)
}

func TestUpdateOrderGroupLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/trade-api/v2/portfolio/order_groups/og-1/limit", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.UpdateOrderGroupLimit(context.Background(), "og-1", UpdateOrderGroupLimitRequest{}, UpdateOrderGroupLimitParams{})
	require.NoError(t, err)
}
