package gokalshi

import (
	"fmt"
	"os"
	"strings"
)

// Environment represents a Kalshi API environment.
type Environment int

const (
	Demo Environment = iota
	Prod
)

func (e Environment) String() string {
	switch e {
	case Demo:
		return "demo"
	case Prod:
		return "prod"
	default:
		return "unknown"
	}
}

const (
	demoHTTPBase = "https://external-api.demo.kalshi.co"
	demoWSBase   = "wss://external-api.demo.kalshi.co"
	prodHTTPBase = "https://external-api.kalshi.com"
	prodWSBase   = "wss://external-api.kalshi.com"
)

// httpBaseForEnv returns the default HTTP base URL for the given environment.
func httpBaseForEnv(env Environment) string {
	switch env {
	case Prod:
		return prodHTTPBase
	case Demo:
		return demoHTTPBase
	default:
		return ""
	}
}

// ClientConfig holds the configuration for connecting to the Kalshi API.
type ClientConfig struct {
	Environment Environment
	Credentials *Credentials
	HTTPBaseURL string
	WSBaseURL   string
}

// NewClientConfig reads environment variables and returns a configured ClientConfig.
func NewClientConfig() (*ClientConfig, error) {
	envStr := strings.ToUpper(os.Getenv("KALSHI_ENVIRONMENT"))
	var env Environment
	switch envStr {
	case "PROD":
		env = Prod
	case "DEMO", "":
		env = Demo
	default:
		return nil, fmt.Errorf("invalid KALSHI_ENVIRONMENT: %q (expected DEMO or PROD)", envStr)
	}

	var keyIDVar, keyFileVar string
	var httpBase, wsBase string

	switch env {
	case Demo:
		keyIDVar = "KALSHI_DEMO_API_KEY_ID"
		keyFileVar = "KALSHI_DEMO_PRIVATE_KEY_FILE"
		httpBase = demoHTTPBase
		wsBase = demoWSBase
	case Prod:
		keyIDVar = "KALSHI_PROD_API_KEY_ID"
		keyFileVar = "KALSHI_PROD_PRIVATE_KEY_FILE"
		httpBase = prodHTTPBase
		wsBase = prodWSBase
	}

	keyID := os.Getenv(keyIDVar)
	keyFile := os.Getenv(keyFileVar)

	if keyID == "" {
		return nil, fmt.Errorf("missing environment variable: %s", keyIDVar)
	}
	if keyFile == "" {
		return nil, fmt.Errorf("missing environment variable: %s", keyFileVar)
	}

	creds, err := LoadCredentials(keyID, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Allow env var overrides for base URLs
	if override := os.Getenv("KALSHI_HTTP_BASE_URL"); override != "" {
		httpBase = override
	}
	if override := os.Getenv("KALSHI_WS_BASE_URL"); override != "" {
		wsBase = override
	}

	return &ClientConfig{
		Environment: env,
		Credentials: creds,
		HTTPBaseURL: httpBase,
		WSBaseURL:   wsBase,
	}, nil
}
