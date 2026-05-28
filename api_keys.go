package gokalshi

import (
	"context"
	"fmt"
)

// GetAPIKeys — Get API Keys
//
// GET /trade-api/v2/api_keys
//
// Endpoint for retrieving all API keys associated with the authenticated user.
// API keys allow programmatic access to the platform without requiring
// username/password authentication. Each key has a unique identifier and name.
//
// See https://trading-api.readme.io/reference/getapikeys
func (c *Client) GetAPIKeys(ctx context.Context) (GetApiKeysResponse, error) {
	return getJSON[GetApiKeysResponse](c, ctx, pathAPIKeys, nil)
}

// CreateAPIKey — Create API Key
//
// POST /trade-api/v2/api_keys
//
// Endpoint for creating a new API key with a user-provided public key. This
// endpoint allows users with Premier or Market Maker API usage levels to
// create API keys by providing their own RSA public key. The platform will use
// this public key to verify signatures on API requests.
//
// See https://trading-api.readme.io/reference/createapikey
func (c *Client) CreateAPIKey(ctx context.Context, req CreateApiKeyRequest) (CreateApiKeyResponse, error) {
	return postJSON[CreateApiKeyResponse](c, ctx, pathAPIKeys, req, 1.0)
}

// GenerateAPIKey — Generate API Key
//
// POST /trade-api/v2/api_keys/generate
//
// Endpoint for generating a new API key with an automatically created key
// pair. This endpoint generates both a public and private RSA key pair. The
// public key is stored on the platform, while the private key is returned to
// the user and must be stored securely. The private key cannot be retrieved
// again.
//
// See https://trading-api.readme.io/reference/generateapikey
func (c *Client) GenerateAPIKey(ctx context.Context, req GenerateApiKeyRequest) (GenerateApiKeyResponse, error) {
	return postJSON[GenerateApiKeyResponse](c, ctx, pathAPIKeys+"/generate", req, 1.0)
}

// DeleteAPIKey — Delete API Key
//
// DELETE /trade-api/v2/api_keys/{api_key}
//
// Endpoint for deleting an existing API key. This endpoint permanently deletes
// an API key. Once deleted, the key can no longer be used for authentication.
// This action cannot be undone.
//
// See https://trading-api.readme.io/reference/deleteapikey
func (c *Client) DeleteAPIKey(ctx context.Context, apiKey string) error {
	path := fmt.Sprintf("%s/%s", pathAPIKeys, apiKey)
	_, err := c.delete(ctx, path, nil, 1.0)
	return err
}
