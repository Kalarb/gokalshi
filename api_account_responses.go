package gokalshi

// Response types for account-related API endpoints.

// GetAccountAPILimitsResponse is the response from GET /account/limits.
type GetAccountAPILimitsResponse struct {
	UsageTier  string `json:"usage_tier"`
	ReadLimit  int    `json:"read_limit"`
	WriteLimit int    `json:"write_limit"`
}
