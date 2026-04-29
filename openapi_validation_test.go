//go:build spec_validation

package gokalshi

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var skippedPathPrefixes = []string{
	"/trade-api/v2/portfolio/subaccounts",
	"/trade-api/v2/portfolio/summary",
	"/trade-api/v2/fcm",
}

func shouldSkipPath(path string) bool {
	for _, prefix := range skippedPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func TestOpenAPICoverage(t *testing.T) {
	resp, err := http.Get("https://docs.kalshi.com/openapi.yaml")
	require.NoError(t, err, "fetch OpenAPI spec")
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, "OpenAPI spec HTTP status")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	require.NoError(t, yaml.Unmarshal(body, &spec))

	type endpoint struct {
		method string
		path   string
	}
	var specEndpoints []endpoint
	for path, methods := range spec.Paths {
		fullPath := path
		if !strings.HasPrefix(path, "/trade-api") {
			fullPath = "/trade-api/v2" + path
		}
		if shouldSkipPath(fullPath) {
			continue
		}
		for method := range methods {
			upper := strings.ToUpper(method)
			if upper == "GET" || upper == "POST" || upper == "PUT" || upper == "DELETE" || upper == "PATCH" {
				specEndpoints = append(specEndpoints, endpoint{upper, fullPath})
			}
		}
	}

	// Count our HTTPClient interface methods (excluding Close which is lifecycle, not an API endpoint).
	httpClientType := reflect.TypeOf((*HTTPClient)(nil)).Elem()
	clientMethods := 0
	for i := 0; i < httpClientType.NumMethod(); i++ {
		name := httpClientType.Method(i).Name
		if name != "Close" {
			clientMethods++
		}
	}

	t.Logf("OpenAPI spec endpoints: %d (after skipping %v)", len(specEndpoints), skippedPathPrefixes)
	t.Logf("HTTPClient interface methods: %d", clientMethods)

	assert.GreaterOrEqual(t, len(specEndpoints), 40, "expected 40+ spec endpoints")
	assert.GreaterOrEqual(t, clientMethods, 36, "expected 36+ client methods")

	// Log all spec endpoints for visibility.
	for _, ep := range specEndpoints {
		t.Logf("  spec: %s %s", ep.method, ep.path)
	}
}
