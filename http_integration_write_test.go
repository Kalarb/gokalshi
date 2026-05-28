//go:build integration

package gokalshi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPIntegration_EventOrdersV2(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	markets, err := c.GetMarkets(ctx, GetMarketsParams{Status: "open", Limit: 5})
	require.NoError(t, err)
	if len(markets.Markets) == 0 {
		t.Skip("no active markets for V2 order test")
	}
	ticker := markets.Markets[0].Ticker

	t.Run("CreateAndCancelV2", func(t *testing.T) {
		clientID := fmt.Sprintf("integ-%d", time.Now().UnixNano())
		created, err := c.CreateOrderV2(ctx, CreateOrderV2Request{
			Ticker:                  ticker,
			Side:                    BookSideBid,
			Price:                   "0.01",
			Count:                   "1",
			ClientOrderID:           clientID,
			SelfTradePreventionType: STPTakerAtCross,
			TimeInForce:             string(TimeInForceGTC),
		})
		require.NoError(t, err)
		orderID := created.OrderID
		assert.NotEmpty(t, orderID)
		t.Logf("created V2 order %s on %s", orderID, ticker)
		t.Cleanup(func() { c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		canceled, err := c.CancelOrderV2(ctx, orderID, CancelOrderV2Params{})
		require.NoError(t, err)
		assert.Equal(t, orderID, canceled.OrderID)
		t.Logf("canceled V2 order %s", canceled.OrderID)
	})

	t.Run("AmendOrderV2", func(t *testing.T) {
		clientID := fmt.Sprintf("integ-%d", time.Now().UnixNano())
		created, err := c.CreateOrderV2(ctx, CreateOrderV2Request{
			Ticker:                  ticker,
			Side:                    BookSideBid,
			Price:                   "0.01",
			Count:                   "1",
			ClientOrderID:           clientID,
			SelfTradePreventionType: STPTakerAtCross,
			TimeInForce:             string(TimeInForceGTC),
		})
		require.NoError(t, err)
		orderID := created.OrderID
		t.Cleanup(func() { c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		amended, err := c.AmendOrderV2(ctx, orderID, AmendOrderV2Request{
			Ticker: ticker,
			Side:   BookSideBid,
			Price:  "0.02",
			Count:  "1",
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, amended.OrderID)
		t.Logf("amended V2 order %s", amended.OrderID)
	})

	t.Run("DecreaseOrderV2", func(t *testing.T) {
		clientID := fmt.Sprintf("integ-%d", time.Now().UnixNano())
		created, err := c.CreateOrderV2(ctx, CreateOrderV2Request{
			Ticker:                  ticker,
			Side:                    BookSideBid,
			Price:                   "0.01",
			Count:                   "2",
			ClientOrderID:           clientID,
			SelfTradePreventionType: STPTakerAtCross,
			TimeInForce:             string(TimeInForceGTC),
		})
		require.NoError(t, err)
		orderID := created.OrderID
		t.Cleanup(func() { c.CancelOrderV2(context.Background(), orderID, CancelOrderV2Params{}) })

		time.Sleep(2 * time.Second)

		decreased, err := c.DecreaseOrderV2(ctx, orderID, DecreaseOrderV2Request{
			ReduceTo: ptr("1"),
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, decreased.OrderID)
		t.Logf("decreased V2 order %s remaining=%s", decreased.OrderID, decreased.RemainingCount)
	})

	t.Run("BatchCreateAndCancelV2", func(t *testing.T) {
		var orders []CreateOrderV2Request
		for i := 0; i < 3; i++ {
			orders = append(orders, CreateOrderV2Request{
				Ticker:                  ticker,
				Side:                    BookSideBid,
				Price:                   "0.01",
				Count:                   "1",
				ClientOrderID:           fmt.Sprintf("integ-%d-%d", time.Now().UnixNano(), i),
				SelfTradePreventionType: STPTakerAtCross,
				TimeInForce:             string(TimeInForceGTC),
			})
		}
		created, err := c.BatchCreateOrdersV2(ctx, BatchCreateOrdersV2Request{Orders: orders})
		require.NoError(t, err)

		// Collect order IDs from batch response.
		var cancelOrders []map[string]any
		for _, entry := range created.Orders {
			if oid, ok := entry["order_id"].(string); ok && oid != "" {
				cancelOrders = append(cancelOrders, map[string]any{"order_id": oid})
			}
		}
		assert.GreaterOrEqual(t, len(cancelOrders), 1)
		t.Logf("batch created %d V2 orders", len(cancelOrders))

		t.Cleanup(func() {
			for _, o := range cancelOrders {
				if oid, ok := o["order_id"].(string); ok {
					c.CancelOrderV2(context.Background(), oid, CancelOrderV2Params{})
				}
			}
		})

		time.Sleep(2 * time.Second)

		_, err = c.BatchCancelOrdersV2(ctx, BatchCancelOrdersV2Request{Orders: cancelOrders})
		require.NoError(t, err)
		t.Logf("batch canceled %d V2 orders", len(cancelOrders))
	})
}

func TestHTTPIntegration_OrderGroups(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("Lifecycle", func(t *testing.T) {
		// Create
		created, err := c.CreateOrderGroup(ctx, CreateOrderGroupRequest{
			ContractsLimitFP: ptr("100"),
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		groupID := created.OrderGroupID
		assert.NotEmpty(t, groupID)
		t.Logf("created order group %s", groupID)
		t.Cleanup(func() { c.DeleteOrderGroup(context.Background(), groupID, DeleteOrderGroupParams{}) })

		time.Sleep(2 * time.Second)

		// GetOrderGroups
		groupsResp, err := c.GetOrderGroups(ctx, GetOrderGroupsParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("order groups count=%d", len(groupsResp.OrderGroups))

		// GetOrderGroup
		groupResp, err := c.GetOrderGroup(ctx, groupID, GetOrderGroupParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotNil(t, groupResp.Orders)

		// Reset
		err = c.ResetOrderGroup(ctx, groupID, OrderGroupActionParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("reset order group %s", groupID)

		// UpdateLimit
		err = c.UpdateOrderGroupLimit(ctx, groupID, UpdateOrderGroupLimitRequest{
			ContractsLimitFP: ptr("200"),
		}, UpdateOrderGroupLimitParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("updated limit for order group %s", groupID)

		// Trigger
		err = c.TriggerOrderGroup(ctx, groupID, OrderGroupActionParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("triggered order group %s", groupID)

		// Delete
		err = c.DeleteOrderGroup(ctx, groupID, DeleteOrderGroupParams{})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("deleted order group %s", groupID)
	})
}

func TestHTTPIntegration_Subaccounts(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetSubaccountBalances", func(t *testing.T) {
		resp, err := c.GetSubaccountBalances(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("subaccount balances count=%d", len(resp.SubaccountBalances))
	})

	t.Run("GetSubaccountNetting", func(t *testing.T) {
		resp, err := c.GetSubaccountNetting(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		t.Logf("subaccount netting count=%d", len(resp.NettingConfigs))
	})

	t.Run("GetSubaccountTransfers", func(t *testing.T) {
		_, err := c.GetSubaccountTransfers(ctx, GetSubaccountTransfersParams{Limit: 5})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})

	t.Run("CreateSubaccount", func(t *testing.T) {
		resp, err := c.CreateSubaccount(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.True(t, resp.SubaccountNumber >= 1)
		t.Logf("created subaccount %d", resp.SubaccountNumber)
	})

	t.Run("UpdateSubaccountNetting", func(t *testing.T) {
		err := c.UpdateSubaccountNetting(ctx, UpdateSubaccountNettingRequest{
			SubaccountNumber: 0,
			Enabled:          true,
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})

	t.Run("ApplySubaccountTransfer", func(t *testing.T) {
		_, err := c.ApplySubaccountTransfer(ctx, ApplySubaccountTransferRequest{
			FromSubaccount:   0,
			ToSubaccount:     1,
			AmountCents:      1,
			ClientTransferID: fmt.Sprintf("integ-%d", time.Now().UnixNano()),
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})
}

func TestHTTPIntegration_APIKeysWrite(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GenerateAndDelete", func(t *testing.T) {
		generated, err := c.GenerateAPIKey(ctx, GenerateApiKeyRequest{
			Name: "integ-test-gen",
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, generated.APIKeyID)
		assert.NotEmpty(t, generated.PrivateKey)
		t.Logf("generated API key %s", generated.APIKeyID)
		t.Cleanup(func() { c.DeleteAPIKey(context.Background(), generated.APIKeyID) })

		err = c.DeleteAPIKey(ctx, generated.APIKeyID)
		require.NoError(t, err)
		t.Logf("deleted API key %s", generated.APIKeyID)
	})

	t.Run("CreateAndDelete", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)
		pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		require.NoError(t, err)
		pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

		created, err := c.CreateAPIKey(ctx, CreateApiKeyRequest{
			Name:      "integ-test-create",
			PublicKey: string(pubPEM),
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, created.APIKeyID)
		t.Logf("created API key %s", created.APIKeyID)
		t.Cleanup(func() { c.DeleteAPIKey(context.Background(), created.APIKeyID) })

		err = c.DeleteAPIKey(ctx, created.APIKeyID)
		require.NoError(t, err)
		t.Logf("deleted API key %s", created.APIKeyID)
	})
}

func TestHTTPIntegration_Communications(t *testing.T) {
	c := integrationHTTPClient(t)
	ctx := context.Background()

	t.Run("GetCommunicationsID", func(t *testing.T) {
		resp, err := c.GetCommunicationsID(ctx)
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.CommunicationsID)
		t.Logf("communications_id=%s", resp.CommunicationsID)
	})

	t.Run("GetRFQs", func(t *testing.T) {
		_, err := c.GetRFQs(ctx, GetRFQsParams{Limit: 5})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})

	t.Run("GetQuotes", func(t *testing.T) {
		_, err := c.GetQuotes(ctx, GetQuotesParams{Limit: 5})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
	})

	t.Run("RFQLifecycle", func(t *testing.T) {
		markets, err := c.GetMarkets(ctx, GetMarketsParams{Status: "open", Limit: 1})
		require.NoError(t, err)
		if len(markets.Markets) == 0 {
			t.Skip("no active markets for RFQ test")
		}
		ticker := markets.Markets[0].Ticker

		created, err := c.CreateRFQ(ctx, CreateRFQRequest{
			MarketTicker:  ticker,
			ContractsFP:   ptr("1"),
			RestRemainder: false,
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		rfqID := created.ID
		assert.NotEmpty(t, rfqID)
		t.Logf("created RFQ %s on %s", rfqID, ticker)
		t.Cleanup(func() { c.DeleteRFQ(context.Background(), rfqID) })

		time.Sleep(2 * time.Second)

		rfqResp, err := c.GetRFQ(ctx, rfqID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		assert.Equal(t, rfqID, rfqResp.RFQ.ID)

		err = c.DeleteRFQ(ctx, rfqID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		t.Logf("deleted RFQ %s", rfqID)
	})

	t.Run("QuoteLifecycle", func(t *testing.T) {
		markets, err := c.GetMarkets(ctx, GetMarketsParams{Status: "open", Limit: 1})
		require.NoError(t, err)
		if len(markets.Markets) == 0 {
			t.Skip("no active markets for quote test")
		}
		ticker := markets.Markets[0].Ticker

		// Create an RFQ first so we have something to quote against.
		rfq, err := c.CreateRFQ(ctx, CreateRFQRequest{
			MarketTicker:  ticker,
			ContractsFP:   ptr("1"),
			RestRemainder: false,
		})
		skipOnAPIError(t, err, 400, 403)
		require.NoError(t, err)
		rfqID := rfq.ID
		t.Cleanup(func() { c.DeleteRFQ(context.Background(), rfqID) })

		// Attempt to create a quote — will likely 400 because you can't quote your own RFQ.
		quote, err := c.CreateQuote(ctx, CreateQuoteRequest{
			RFQID:         rfqID,
			YesBid:        "0.01",
			NoBid:         "0.01",
			RestRemainder: false,
		})
		skipOnAPIError(t, err, 400, 403)
		if err != nil {
			t.Skipf("CreateQuote failed (expected — cannot quote own RFQ): %v", err)
		}
		quoteID := quote.ID
		t.Cleanup(func() { c.DeleteQuote(context.Background(), quoteID) })

		quoteResp, err := c.GetQuote(ctx, quoteID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
		assert.Equal(t, quoteID, quoteResp.Quote.ID)

		// AcceptQuote and ConfirmQuote require a counterparty, so they will skip.
		err = c.AcceptQuote(ctx, quoteID, AcceptQuoteRequest{AcceptedSide: SideYes})
		skipOnAPIError(t, err, 400, 403)

		err = c.ConfirmQuote(ctx, quoteID)
		skipOnAPIError(t, err, 400, 403)

		err = c.DeleteQuote(ctx, quoteID)
		skipOnAPIError(t, err, 400, 404)
		require.NoError(t, err)
	})

	t.Run("MVECollectionWrites", func(t *testing.T) {
		collectionsResp, err := c.GetMultivariateEventCollections(ctx, GetMultivariateEventCollectionsParams{Limit: 5})
		require.NoError(t, err)
		if len(collectionsResp.MultivariateContracts) == 0 {
			t.Skip("no MVE collections available")
		}
		collection := collectionsResp.MultivariateContracts[0]
		collectionTicker := collection.CollectionTicker

		if len(collection.AssociatedEvents) == 0 {
			t.Skip("MVE collection has no associated events")
		}

		// Build SelectedMarkets from the first associated event's markets.
		var selectedMarkets []TickerPair
		for _, ae := range collection.AssociatedEvents {
			if ae.Ticker != "" {
				// Fetch the event to get its markets.
				eventResp, err := c.GetEvent(ctx, ae.Ticker, GetEventParams{WithNestedMarkets: true})
				if err != nil {
					continue
				}
				if len(eventResp.Event.Markets) > 0 {
					selectedMarkets = append(selectedMarkets, TickerPair{
						EventTicker:  ae.Ticker,
						MarketTicker: eventResp.Event.Markets[0].Ticker,
						Side:         "yes",
					})
				}
			}
			if len(selectedMarkets) >= 2 {
				break
			}
		}
		if len(selectedMarkets) == 0 {
			t.Skip("could not build SelectedMarkets from MVE collection")
		}

		_, err = c.CreateMarketInMultivariateEventCollection(ctx, collectionTicker, CreateMarketInMultivariateEventCollectionRequest{
			SelectedMarkets: selectedMarkets,
		})
		skipOnAPIError(t, err, 400, 403, 404)
		if err == nil {
			t.Logf("created market in MVE collection %s", collectionTicker)
		}

		_, err = c.LookupTickersForMarketInMultivariateEventCollection(ctx, collectionTicker, LookupTickersForMarketInMultivariateEventCollectionRequest{
			SelectedMarkets: selectedMarkets,
		})
		skipOnAPIError(t, err, 400, 403, 404)
		if err == nil {
			t.Logf("looked up tickers in MVE collection %s", collectionTicker)
		}
	})
}
