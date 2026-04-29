package gokalshi

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempPEM(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	pemPath := filepath.Join(t.TempDir(), "test_key.pem")
	f, err := os.Create(pemPath)
	require.NoError(t, err)
	defer f.Close()

	err = pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der})
	require.NoError(t, err)

	return pemPath
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestEnvironmentString(t *testing.T) {
	assert.Equal(t, "demo", Demo.String())
	assert.Equal(t, "prod", Prod.String())
	assert.Equal(t, "unknown", Environment(99).String())
}

func TestNewClientConfig_Demo(t *testing.T) {
	pemPath := writeTempPEM(t)

	setEnv(t, "KALSHI_ENVIRONMENT", "DEMO")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "test-key")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", pemPath)

	cfg, err := NewClientConfig()
	require.NoError(t, err)

	assert.Equal(t, Demo, cfg.Environment)
	assert.Equal(t, "test-key", cfg.Credentials.KeyID)
	assert.Equal(t, demoHTTPBase, cfg.HTTPBaseURL)
	assert.Equal(t, demoWSBase, cfg.WSBaseURL)
}

func TestNewClientConfig_Prod(t *testing.T) {
	pemPath := writeTempPEM(t)

	setEnv(t, "KALSHI_ENVIRONMENT", "PROD")
	setEnv(t, "KALSHI_PROD_API_KEY_ID", "prod-key")
	setEnv(t, "KALSHI_PROD_PRIVATE_KEY_FILE", pemPath)

	cfg, err := NewClientConfig()
	require.NoError(t, err)

	assert.Equal(t, Prod, cfg.Environment)
	assert.Equal(t, "prod-key", cfg.Credentials.KeyID)
	assert.Equal(t, prodHTTPBase, cfg.HTTPBaseURL)
	assert.Equal(t, prodWSBase, cfg.WSBaseURL)
}

func TestNewClientConfig_DefaultsToDemo(t *testing.T) {
	pemPath := writeTempPEM(t)

	setEnv(t, "KALSHI_ENVIRONMENT", "")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "test-key")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", pemPath)

	cfg, err := NewClientConfig()
	require.NoError(t, err)

	assert.Equal(t, Demo, cfg.Environment)
}

func TestNewClientConfig_URLOverrides(t *testing.T) {
	pemPath := writeTempPEM(t)

	setEnv(t, "KALSHI_ENVIRONMENT", "DEMO")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "test-key")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", pemPath)
	setEnv(t, "KALSHI_HTTP_BASE_URL", "http://localhost:8080")
	setEnv(t, "KALSHI_WS_BASE_URL", "ws://localhost:8081")

	cfg, err := NewClientConfig()
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:8080", cfg.HTTPBaseURL)
	assert.Equal(t, "ws://localhost:8081", cfg.WSBaseURL)
}

func TestNewClientConfig_InvalidEnvironment(t *testing.T) {
	setEnv(t, "KALSHI_ENVIRONMENT", "STAGING")

	_, err := NewClientConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid KALSHI_ENVIRONMENT")
}

func TestNewClientConfig_MissingKeyID(t *testing.T) {
	setEnv(t, "KALSHI_ENVIRONMENT", "DEMO")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", "/some/path")

	_, err := NewClientConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing environment variable")
}

func TestNewClientConfig_MissingKeyFile(t *testing.T) {
	setEnv(t, "KALSHI_ENVIRONMENT", "DEMO")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "test-key")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", "")

	_, err := NewClientConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing environment variable")
}

func TestNewClientConfig_BadKeyFile(t *testing.T) {
	setEnv(t, "KALSHI_ENVIRONMENT", "DEMO")
	setEnv(t, "KALSHI_DEMO_API_KEY_ID", "test-key")
	setEnv(t, "KALSHI_DEMO_PRIVATE_KEY_FILE", "/nonexistent/key.pem")

	_, err := NewClientConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load credentials")
}
