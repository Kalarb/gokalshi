package gokalshi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// ptr returns a pointer to the given value. Used in test structs for *string fields etc.
func ptr[T any](v T) *T { return &v }

// loadEnv walks up from CWD to find go.mod, then loads .env from that directory.
// No-op if .env doesn't exist (env vars may be set via CI/shell).
func loadEnv(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

// skipWithoutCreds skips the test if the given env vars are missing or the key file doesn't exist.
// Returns (keyID, keyFile) if credentials are available.
func skipWithoutCreds(t *testing.T, keyIDVar, keyFileVar string) (string, string) {
	t.Helper()
	loadEnv(t)
	keyID := os.Getenv(keyIDVar)
	keyFile := os.Getenv(keyFileVar)
	if keyID == "" || keyFile == "" {
		t.Skipf("skipping: %s or %s not set", keyIDVar, keyFileVar)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Skipf("skipping: key file %s does not exist", keyFile)
	}
	return keyID, keyFile
}

// integrationHTTPClient creates a DEMO *Client from env vars.
func integrationHTTPClient(t *testing.T) *Client {
	t.Helper()
	keyID, keyFile := skipWithoutCreds(t, "KALSHI_DEMO_API_KEY_ID", "KALSHI_DEMO_PRIVATE_KEY_FILE")
	creds, err := LoadCredentials(keyID, keyFile)
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	cfg := &ClientConfig{
		Environment: Demo,
		Credentials: creds,
		HTTPBaseURL: demoHTTPBase,
		WSBaseURL:   demoWSBase,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

// integrationProdHTTPClient creates a PROD *Client from read-only env vars.
func integrationProdHTTPClient(t *testing.T) *Client {
	t.Helper()
	keyID, keyFile := skipWithoutCreds(t, "KALSHI_PROD_READ_ONLY_API_KEY_ID", "KALSHI_PROD_READ_ONLY_PRIVATE_KEY_FILE")
	creds, err := LoadCredentials(keyID, keyFile)
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	cfg := &ClientConfig{
		Environment: Prod,
		Credentials: creds,
		HTTPBaseURL: prodHTTPBase,
		WSBaseURL:   prodWSBase,
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	t.Cleanup(c.Close)
	return c
}

// integrationWSClient creates a PROD *WSClient from read-only env vars.
func integrationWSClient(t *testing.T) *WSClient {
	t.Helper()
	keyID, keyFile := skipWithoutCreds(t, "KALSHI_PROD_READ_ONLY_API_KEY_ID", "KALSHI_PROD_READ_ONLY_PRIVATE_KEY_FILE")
	creds, err := LoadCredentials(keyID, keyFile)
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	cfg := &ClientConfig{
		Environment: Prod,
		Credentials: creds,
		HTTPBaseURL: prodHTTPBase,
		WSBaseURL:   prodWSBase,
	}
	ws := NewWSClient(cfg)
	t.Cleanup(func() { ws.Close() })
	return ws
}

// newIntegrationOrder builds a minimal resting V2 order: priced far from the
// market so it rests rather than filling, which keeps order-lifecycle tests
// from accidentally trading.
func newIntegrationOrder(ticker, price, count string) CreateOrderV2Request {
	return CreateOrderV2Request{
		Ticker:                  ticker,
		Side:                    BookSideBid,
		Count:                   count,
		Price:                   price,
		TimeInForce:             TimeInForceGTC,
		SelfTradePreventionType: STPTakerAtCross,
	}
}

// weatherCityCandidates are probed in order by weatherCity.
//
// The API has no endpoint that enumerates weather index cities and the spec
// names only `miami` as an example, so there is nothing to discover from. These
// are probed rather than assumed: the first that answers is used, and the test
// skips if none do, so Kalshi adding or retiring a city degrades to a skip
// rather than a failure.
var weatherCityCandidates = []string{"miami", "nyc", "chicago", "austin", "denver", "philadelphia", "la"}

// weatherCity returns a city id whose weather index this account can read.
func weatherCity(t *testing.T, c *Client, ctx context.Context) string {
	t.Helper()
	for _, city := range weatherCityCandidates {
		if _, err := c.GetWeatherIndexCalibrations(ctx, city); err == nil {
			t.Logf("using weather city %q", city)
			return city
		}
	}
	t.Skipf("no weather index city responded (probed %v)", weatherCityCandidates)
	return ""
}

// liveDataSeries are series whose events carry live data — crypto price charts,
// commodity timeseries and weather observations, per the spec. Probed in order.
var liveDataSeries = []string{"KXBTCD", "KXBTC15M", "KXETHD"}

// liveDataEvent returns an event ticker that actually has live data attached.
// Most events do not, so GetEventLiveData 404s on a randomly chosen one.
func liveDataEvent(t *testing.T, c *Client, ctx context.Context) string {
	t.Helper()
	for _, series := range liveDataSeries {
		events, err := c.GetEvents(ctx, GetEventsParams{
			Status: "open", SeriesTicker: series, Limit: 5,
		})
		if err != nil || len(events.Events) == 0 {
			continue
		}
		for _, e := range events.Events {
			if _, err := c.GetEventLiveData(ctx, e.EventTicker, GetEventLiveDataParams{}); err == nil {
				t.Logf("using live-data event %s from series %s", e.EventTicker, series)
				return e.EventTicker
			}
		}
	}
	t.Skipf("no event with live data found (probed series %v)", liveDataSeries)
	return ""
}
