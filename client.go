package gokalshi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Client is the Kalshi HTTP API client.
// It wraps net/http with authentication, rate limiting, and retry logic.
// On creation, it fetches account rate limits and endpoint costs from the
// API to configure the rate limiter automatically.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	credentials    *Credentials
	maxRetries     int
	baseDelay      time.Duration
	skipAutoConfig bool

	mu          sync.RWMutex // guards limiter, costRoutes, defaultCost
	limiter     *ReadWriteTokenBucket
	costRoutes  []costRoute // nil = use caller defaults
	defaultCost float64     // fallback cost when no route matches; 0 = use caller defaults
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cl *Client) { cl.httpClient = c }
}

// WithMaxRetries sets the max number of 429 retries.
func WithMaxRetries(n int) ClientOption {
	return func(cl *Client) { cl.maxRetries = n }
}

// WithBaseDelay sets the base retry delay for 429 responses.
func WithBaseDelay(d time.Duration) ClientOption {
	return func(cl *Client) { cl.baseDelay = d }
}

// WithRateLimiter sets a custom rate limiter and disables auto-configuration.
// Use this to opt out of the automatic rate limit fetch at startup.
func WithRateLimiter(l *ReadWriteTokenBucket) ClientOption {
	return func(cl *Client) {
		cl.limiter = l
		cl.skipAutoConfig = true
	}
}

// NewClient creates a new Kalshi HTTP client from a ClientConfig.
// It fetches account rate limits and endpoint costs from the API to
// configure the rate limiter automatically. To skip auto-configuration,
// pass WithRateLimiter with a custom limiter.
func NewClient(cfg *ClientConfig, opts ...ClientOption) (*Client, error) {
	baseURL := cfg.HTTPBaseURL
	if baseURL == "" {
		baseURL = httpBaseForEnv(cfg.Environment)
	}

	c := &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		baseURL:     baseURL,
		credentials: cfg.Credentials,
		limiter:     NewReadWriteTokenBucket(DefaultTokenBucketConfig()),
		maxRetries:  4,
		baseDelay:   100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(c)
	}

	if !c.skipAutoConfig {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := c.ConfigureRateLimits(ctx); err != nil {
			log.Printf("gokalshi: auto-configure rate limits failed (using defaults): %v", err)
		}
	}

	return c, nil
}

// Close releases resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// ConfigureRateLimits fetches account API limits and endpoint costs from the
// Kalshi API and configures the client's rate limiter and cost map accordingly.
// Called automatically during NewClient. Can be called again to refresh.
// Safe for concurrent use — holds no lock during API calls, applies changes atomically.
func (c *Client) ConfigureRateLimits(ctx context.Context) error {
	limits, err := c.GetAccountAPILimits(ctx)
	if err != nil {
		return fmt.Errorf("fetch account limits: %w", err)
	}

	newLimiter := NewReadWriteTokenBucket(TokenBucketConfig{
		ReadRate:      float64(limits.Read.RefillRate),
		WriteRate:     float64(limits.Write.RefillRate),
		ReadCapacity:  float64(limits.Read.BucketCapacity),
		WriteCapacity: float64(limits.Write.BucketCapacity),
		WindowSize:    1.0,
		SafetyPadding: 0.1,
	})

	endpointCosts, err := c.GetAccountEndpointCosts(ctx)
	if err != nil {
		return fmt.Errorf("fetch endpoint costs: %w", err)
	}

	routes := make([]costRoute, 0, len(endpointCosts.EndpointCosts))
	for _, ec := range endpointCosts.EndpointCosts {
		routes = append(routes, newCostRoute(ec.Method, ec.Path, float64(ec.Cost)))
	}

	// Most specific first, so a literal route wins over one whose placeholder
	// would also match. Without this, "/portfolio/events/orders/:order_id"
	// could claim "/portfolio/events/orders/batched" purely by declaration
	// order, and the two do not have to share a cost.
	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].literals > routes[j].literals
	})

	// A route this code cannot match does not fail — it falls through to the
	// default cost, which is indistinguishable from an endpoint that has no
	// override. Say so, so the next syntax change is a log line rather than a
	// year of quietly mis-billed cancels.
	for _, r := range routes {
		if suspects := r.suspiciousSegments(); len(suspects) > 0 {
			log.Printf("gokalshi: endpoint cost %s %s has segments %v that look like "+
				"unrecognised path parameters — this entry may never match, and its "+
				"endpoint will bill at the default cost of %.0f",
				r.method, r.rawPath, suspects, float64(endpointCosts.DefaultCost))
		}
	}

	c.mu.Lock()
	c.limiter = newLimiter
	c.costRoutes = routes
	c.defaultCost = float64(endpointCosts.DefaultCost)
	c.mu.Unlock()

	return nil
}

// resolveCosts returns the effective read/write costs for a request.
// If a cost pattern matches, it overrides the caller's values.
// Must be called with c.mu held for reading (or from a non-concurrent context).
// resolveCosts determines what to debit from the rate limiter for one request.
//
// The server's endpoint-costs table is authoritative for the cost of a single
// request, so a matching route replaces the caller's figure. The caller keeps
// ownership of units: batch endpoints are billed per item, and the table has no
// way to say so — EndpointTokenCost is a flat integer. Previously the resolved
// cost replaced the total, which discarded the multiplication a batch call site
// had already done, and a fifty-order batch was billed as one request.
func (c *Client) resolveCosts(method, path string, readCost, writeCost float64, units int) (float64, float64) {
	if units < 1 {
		units = 1
	}
	// The caller's figures are per unit, exactly like the table's, so every
	// path scales by the same count. Returning them unscaled here would mean
	// writeCost silently changed meaning — per-unit with a cost table loaded,
	// total without one — and which applied depended on whether an HTTP call
	// at startup had succeeded.
	if c.costRoutes == nil {
		return readCost * float64(units), writeCost * float64(units)
	}

	unit, matched := c.lookupCost(method, path)
	if !matched {
		if c.defaultCost <= 0 {
			return readCost * float64(units), writeCost * float64(units)
		}
		unit = c.defaultCost
	}

	cost := unit * float64(units)
	if method == http.MethodGet {
		return cost, 0
	}
	return 0, cost
}

// lookupCost returns the per-request cost for an endpoint, if the table
// overrides it.
func (c *Client) lookupCost(method, path string) (float64, bool) {
	for _, r := range c.costRoutes {
		if r.matches(method, path) {
			return r.cost, true
		}
	}
	return 0, false
}

// do executes an HTTP request with rate limiting, auth headers, and 429 retry.
func (c *Client) do(ctx context.Context, method, path string, readCost, writeCost float64, units int, body any, params map[string]string) (json.RawMessage, error) {
	c.mu.RLock()
	readCost, writeCost = c.resolveCosts(method, path, readCost, writeCost, units)
	limiter := c.limiter
	c.mu.RUnlock()

	if err := limiter.Acquire(ctx, readCost, writeCost); err != nil {
		return nil, fmt.Errorf("rate limit acquire: %w", err)
	}

	fullURL := buildURL(c.baseURL, path, params)

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
	}

	for retries := 0; ; retries++ {
		req, err := c.buildAuthRequest(ctx, method, path, fullURL, bodyBytes)
		if err != nil {
			return nil, err
		}

		respBody, statusCode, err := c.executeRequest(req)
		if err != nil {
			return nil, fmt.Errorf("HTTP %s %s: %w", method, path, err)
		}

		if statusCode == http.StatusTooManyRequests {
			if retried, err := c.handleRetry(ctx, method, path, retries); err != nil {
				return nil, err
			} else if retried {
				continue
			}
		}

		if statusCode >= 400 {
			return nil, newAPIError(statusCode, method, path, string(respBody))
		}

		return json.RawMessage(respBody), nil
	}
}

// buildURL constructs the full URL with query parameters.
func buildURL(baseURL, path string, params map[string]string) string {
	fullURL := baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			if v != "" {
				q.Set(k, v)
			}
		}
		fullURL += "?" + q.Encode()
	}
	return fullURL
}

// buildAuthRequest creates an authenticated HTTP request.
func (c *Client) buildAuthRequest(ctx context.Context, method, path, fullURL string, bodyBytes []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	headers, err := c.credentials.RequestHeaders(method, path)
	if err != nil {
		return nil, fmt.Errorf("generate auth headers: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// executeRequest sends the request and returns the response body and status code.
func (c *Client) executeRequest(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}
	return body, resp.StatusCode, nil
}

// handleRetry handles 429 rate limit responses with exponential backoff.
// Returns (true, nil) if a retry should be attempted, or (false, error) if exhausted/cancelled.
func (c *Client) handleRetry(ctx context.Context, method, path string, retries int) (bool, error) {
	if retries >= c.maxRetries {
		return false, &RateLimitError{Method: method, Path: path, Retries: retries}
	}

	shift := retries
	if shift > 30 {
		shift = 30
	}
	wait := c.baseDelay * (1 << shift)

	select {
	case <-ctx.Done():
		return false, fmt.Errorf("retry cancelled: %w", ctx.Err())
	case <-time.After(wait):
		return true, nil
	}
}

// doJSON executes an HTTP request and unmarshals the response into T.
func doJSON[T any](c *Client, ctx context.Context, method, path string, readCost, writeCost float64, units int, body any, params map[string]string) (T, error) {
	raw, err := c.do(ctx, method, path, readCost, writeCost, units, body, params)
	var result T
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("unmarshal %s %s response: %w", method, path, err)
	}
	return result, nil
}

// Convenience methods for HTTP verbs.

func (c *Client) get(ctx context.Context, path string, params map[string]string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, path, 10.0, 0, 1, nil, params)
}

func getJSON[T any](c *Client, ctx context.Context, path string, params map[string]string) (T, error) {
	return doJSON[T](c, ctx, http.MethodGet, path, 10.0, 0, 1, nil, params)
}

func (c *Client) post(ctx context.Context, path string, body any, writeCost float64) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, 0, writeCost, 1, body, nil)
}

func postJSON[T any](c *Client, ctx context.Context, path string, body any, writeCost float64) (T, error) {
	return doJSON[T](c, ctx, http.MethodPost, path, 0, writeCost, 1, body, nil)
}

func (c *Client) delete(ctx context.Context, path string, body any, writeCost float64) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, path, 0, writeCost, 1, body, nil)
}

func (c *Client) put(ctx context.Context, path string, body any, writeCost float64) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPut, path, 0, writeCost, 1, body, nil)
}
