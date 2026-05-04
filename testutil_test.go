package gokalshi

import (
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
	c := NewClient(cfg)
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
	c := NewClient(cfg)
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
