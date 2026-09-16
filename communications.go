package gokalshi

import (
	"context"
	"fmt"
)

// --- Communications ID ---

// GetCommunicationsID — Get Communications ID
//
// GET /trade-api/v2/communications/id
//
// Endpoint for getting the communications ID of the logged-in user.
//
// See https://docs.kalshi.com/api-reference/communications/get-communications-i-d
func (c *Client) GetCommunicationsID(ctx context.Context) (GetCommunicationsIDResponse, error) {
	return getJSON[GetCommunicationsIDResponse](c, ctx, pathCommunications+"/id", nil)
}

// --- RFQs ---

// CreateRFQ — Create RFQ
//
// POST /trade-api/v2/communications/rfqs
//
// Endpoint for creating a new RFQ. You can have a maximum of 100 open RFQs at
// a time.
//
// See https://docs.kalshi.com/api-reference/communications/create-r-f-q
func (c *Client) CreateRFQ(ctx context.Context, req CreateRFQRequest) (CreateRFQResponse, error) {
	return postJSON[CreateRFQResponse](c, ctx, pathCommunications+"/rfqs", req, 10.0)
}

// GetRFQs — Get RFQs
//
// GET /trade-api/v2/communications/rfqs
//
// # Endpoint for getting RFQs
//
// See https://docs.kalshi.com/api-reference/communications/get-r-f-qs
func (c *Client) GetRFQs(ctx context.Context, params GetRFQsParams) (GetRFQsResponse, error) {
	return getJSON[GetRFQsResponse](c, ctx, pathCommunications+"/rfqs", params.toMap())
}

// GetRFQ — Get RFQ
//
// GET /trade-api/v2/communications/rfqs/{rfq_id}
//
// # Endpoint for getting a single RFQ by id
//
// See https://docs.kalshi.com/api-reference/communications/get-r-f-q
func (c *Client) GetRFQ(ctx context.Context, rfqID string) (GetRFQResponse, error) {
	path := fmt.Sprintf("%s/rfqs/%s", pathCommunications, rfqID)
	return getJSON[GetRFQResponse](c, ctx, path, nil)
}

// DeleteRFQ — Delete RFQ
//
// DELETE /trade-api/v2/communications/rfqs/{rfq_id}
//
// # Endpoint for deleting an RFQ by ID
//
// See https://docs.kalshi.com/api-reference/communications/delete-r-f-q
func (c *Client) DeleteRFQ(ctx context.Context, rfqID string) error {
	path := fmt.Sprintf("%s/rfqs/%s", pathCommunications, rfqID)
	_, err := c.delete(ctx, path, nil, 10.0)
	return err
}

// GetRFQsParams are query parameters for GetRFQs.
type GetRFQsParams struct {
	Cursor      string
	EventTicker string
	Ticker      string
	Subaccount  string
	Limit       int
	Status      string
	UserFilter  string
}

func (p GetRFQsParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		String("event_ticker", p.EventTicker).
		String("market_ticker", p.Ticker).
		String("subaccount", p.Subaccount).
		Int("limit", p.Limit).
		String("status", p.Status).
		String("user_filter", p.UserFilter).
		Build()
}

// --- Quotes ---

// CreateQuote — Create Quote
//
// POST /trade-api/v2/communications/quotes
//
// # Endpoint for creating a quote in response to an RFQ
//
// See https://docs.kalshi.com/api-reference/communications/create-quote
func (c *Client) CreateQuote(ctx context.Context, req CreateQuoteRequest) (CreateQuoteResponse, error) {
	return postJSON[CreateQuoteResponse](c, ctx, pathCommunications+"/quotes", req, 10.0)
}

// GetQuotes — Get Quotes
//
// GET /trade-api/v2/communications/quotes
//
// # Endpoint for getting quotes
//
// See https://docs.kalshi.com/api-reference/communications/get-quotes
func (c *Client) GetQuotes(ctx context.Context, params GetQuotesParams) (GetQuotesResponse, error) {
	return getJSON[GetQuotesResponse](c, ctx, pathCommunications+"/quotes", params.toMap())
}

// GetQuote — Get Quote
//
// GET /trade-api/v2/communications/quotes/{quote_id}
//
// DEPRECATED: Use GET /communications/rfqs/{rfq_id}/quotes/{quote_id} instead.
// Endpoint for getting a particular quote.
//
// Deprecated: Kalshi marks the quote-id-only form deprecated. Use GetRFQQuote,
// which scopes the quote to its RFQ.
//
// See https://docs.kalshi.com/api-reference/communications/get-quote
func (c *Client) GetQuote(ctx context.Context, quoteID string) (GetQuoteResponse, error) {
	path := fmt.Sprintf("%s/quotes/%s", pathCommunications, quoteID)
	return getJSON[GetQuoteResponse](c, ctx, path, nil)
}

// DeleteQuote — Delete Quote
//
// DELETE /trade-api/v2/communications/quotes/{quote_id}
//
// DEPRECATED: Use DELETE /communications/rfqs/{rfq_id}/quotes/{quote_id}
// instead. Endpoint for deleting a quote, which means it can no longer be
// accepted.
//
// Deprecated: Kalshi marks the quote-id-only form deprecated. Use DeleteRFQQuote,
// which scopes the quote to its RFQ.
//
// See https://docs.kalshi.com/api-reference/communications/delete-quote
func (c *Client) DeleteQuote(ctx context.Context, quoteID string) error {
	path := fmt.Sprintf("%s/quotes/%s", pathCommunications, quoteID)
	_, err := c.delete(ctx, path, nil, 10.0)
	return err
}

// AcceptQuote — Accept Quote
//
// PUT /trade-api/v2/communications/quotes/{quote_id}/accept
//
// DEPRECATED: Use PUT /communications/rfqs/{rfq_id}/quotes/{quote_id}/accept
// instead. Endpoint for accepting a quote. This will require the quoter to
// confirm.
//
// Deprecated: Kalshi marks the quote-id-only form deprecated. Use AcceptRFQQuote,
// which scopes the quote to its RFQ.
//
// See https://docs.kalshi.com/api-reference/communications/accept-quote
func (c *Client) AcceptQuote(ctx context.Context, quoteID string, req AcceptQuoteRequest) error {
	path := fmt.Sprintf("%s/quotes/%s/accept", pathCommunications, quoteID)
	_, err := c.put(ctx, path, req, 10.0)
	return err
}

// ConfirmQuote — Confirm Quote
//
// PUT /trade-api/v2/communications/quotes/{quote_id}/confirm
//
// DEPRECATED: Use PUT /communications/rfqs/{rfq_id}/quotes/{quote_id}/confirm
// instead. Endpoint for confirming a quote. This will start a timer for order
// execution.
//
// Deprecated: Kalshi marks the quote-id-only form deprecated. Use ConfirmRFQQuote,
// which scopes the quote to its RFQ.
//
// See https://docs.kalshi.com/api-reference/communications/confirm-quote
func (c *Client) ConfirmQuote(ctx context.Context, quoteID string) error {
	path := fmt.Sprintf("%s/quotes/%s/confirm", pathCommunications, quoteID)
	_, err := c.put(ctx, path, nil, 10.0)
	return err
}

// GetQuotesParams are query parameters for GetQuotes.
type GetQuotesParams struct {
	Cursor        string
	EventTicker   string
	Ticker        string
	Limit         int
	Status        string
	UserFilter    string
	RFQUserFilter string
	RFQID         string
}

func (p GetQuotesParams) toMap() map[string]string {
	return NewQuery().
		String("cursor", p.Cursor).
		String("event_ticker", p.EventTicker).
		String("market_ticker", p.Ticker).
		Int("limit", p.Limit).
		String("status", p.Status).
		String("user_filter", p.UserFilter).
		String("rfq_user_filter", p.RFQUserFilter).
		String("rfq_id", p.RFQID).
		Build()
}

// GetRFQQuote — Get RFQ Quote
//
// GET /trade-api/v2/communications/rfqs/{rfq_id}/quotes/{quote_id}
//
// Endpoint for getting a particular quote scoped to its RFQ.
//
// See https://docs.kalshi.com/api-reference/communications/get-r-f-q-quote
func (c *Client) GetRFQQuote(ctx context.Context, rfqID, quoteID string) (GetQuoteResponse, error) {
	return doJSON[GetQuoteResponse](c, ctx, "GET", rfqQuotePath(rfqID, quoteID), 2.0, 0, 1, nil, nil)
}

// DeleteRFQQuote — Delete RFQ Quote
//
// DELETE /trade-api/v2/communications/rfqs/{rfq_id}/quotes/{quote_id}
//
// Endpoint for deleting a quote scoped to its RFQ, which means it can no
// longer be accepted.
//
// See https://docs.kalshi.com/api-reference/communications/delete-r-f-q-quote
func (c *Client) DeleteRFQQuote(ctx context.Context, rfqID, quoteID string) error {
	_, err := c.do(ctx, "DELETE", rfqQuotePath(rfqID, quoteID), 0, 2.0, 1, nil, nil)
	return err
}

// AcceptRFQQuote — Accept RFQ Quote
//
// PUT /trade-api/v2/communications/rfqs/{rfq_id}/quotes/{quote_id}/accept
//
// Endpoint for accepting a quote scoped to its RFQ. This will require the
// quoter to confirm.
//
// See https://docs.kalshi.com/api-reference/communications/accept-r-f-q-quote
func (c *Client) AcceptRFQQuote(ctx context.Context, rfqID, quoteID string, req AcceptQuoteRequest) error {
	_, err := c.put(ctx, rfqQuotePath(rfqID, quoteID)+"/accept", req, 10.0)
	return err
}

// ConfirmRFQQuote — Confirm RFQ Quote
//
// PUT /trade-api/v2/communications/rfqs/{rfq_id}/quotes/{quote_id}/confirm
//
// Endpoint for confirming a quote scoped to its RFQ. This will start a timer
// for order execution.
//
// See https://docs.kalshi.com/api-reference/communications/confirm-r-f-q-quote
func (c *Client) ConfirmRFQQuote(ctx context.Context, rfqID, quoteID string) error {
	_, err := c.put(ctx, rfqQuotePath(rfqID, quoteID)+"/confirm", struct{}{}, 10.0)
	return err
}

func rfqQuotePath(rfqID, quoteID string) string {
	return fmt.Sprintf("%s/rfqs/%s/quotes/%s", pathCommunications, rfqID, quoteID)
}
