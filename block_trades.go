package gokalshi

import (
	"context"
	"fmt"
)

// GetBlockTradeProposals — Get Block Trade Proposals
//
// GET /trade-api/v2/communications/block-trade-proposals
//
// Endpoint for getting block trade proposals visible to the authenticated
// user.
//
// See https://docs.kalshi.com/api-reference/communications/get-block-trade-proposals
func (c *Client) GetBlockTradeProposals(ctx context.Context, params GetBlockTradeProposalsParams) (GetBlockTradeProposalsResponse, error) {
	return getJSON[GetBlockTradeProposalsResponse](c, ctx, pathBlockTradeProposals, params.toMap())
}

// GetBlockTradeProposalsParams are the query parameters for GetBlockTradeProposals.
type GetBlockTradeProposalsParams struct {
	MarketTicker string
	Status       string
	Limit        int
	Cursor       string
}

func (p GetBlockTradeProposalsParams) toMap() map[string]string {
	return NewQuery().
		String("market_ticker", p.MarketTicker).
		String("status", p.Status).
		Int("limit", p.Limit).
		String("cursor", p.Cursor).
		Build()
}

// ProposeBlockTrade — Propose Block Trade
//
// POST /trade-api/v2/communications/block-trade-proposals
//
// Endpoint for creating a block trade proposal.
//
// See https://docs.kalshi.com/api-reference/communications/propose-block-trade
func (c *Client) ProposeBlockTrade(ctx context.Context, req ProposeBlockTradeRequest) (ProposeBlockTradeResponse, error) {
	return postJSON[ProposeBlockTradeResponse](c, ctx, pathBlockTradeProposals, req, 10.0)
}

// AcceptBlockTradeProposal — Accept Block Trade Proposal
//
// POST /trade-api/v2/communications/block-trade-proposals/{block_trade_proposal_id}/accept
//
// Endpoint for accepting a block trade proposal.
//
// See https://docs.kalshi.com/api-reference/communications/accept-block-trade-proposal
func (c *Client) AcceptBlockTradeProposal(ctx context.Context, proposalID string, req AcceptBlockTradeProposalRequest) error {
	path := fmt.Sprintf("%s/%s/accept", pathBlockTradeProposals, proposalID)
	_, err := c.post(ctx, path, req, 10.0)
	return err
}
