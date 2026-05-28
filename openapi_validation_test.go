//go:build spec_validation

package gokalshi

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var skippedPathPrefixes = []string{
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

// endpointRe matches godoc comments like "// GET /trade-api/v2/markets".
var endpointRe = regexp.MustCompile(`^//\s+(GET|POST|PUT|DELETE|PATCH)\s+(/trade-api/v2\S+)`)

// parseImplementedEndpoints scans all .go source files (excluding tests) for
// godoc comments containing "METHOD /trade-api/v2/..." and returns the set of
// implemented endpoints as "METHOD /path" strings.
func parseImplementedEndpoints(t *testing.T) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	implemented := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		require.NoError(t, err)

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if m := endpointRe.FindStringSubmatch(line); m != nil {
				key := m[1] + " " + m[2]
				implemented[key] = true
			}
		}
	}
	return implemented
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

	// Build set of spec endpoints.
	specEndpoints := make(map[string]bool)
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
				specEndpoints[upper+" "+fullPath] = true
			}
		}
	}

	// Parse implemented endpoints from godoc comments.
	implemented := parseImplementedEndpoints(t)

	// Find spec endpoints not implemented in the SDK.
	var missing []string
	for ep := range specEndpoints {
		if !implemented[ep] {
			missing = append(missing, ep)
		}
	}
	sort.Strings(missing)

	t.Logf("OpenAPI spec endpoints: %d (after skipping %v)", len(specEndpoints), skippedPathPrefixes)
	t.Logf("SDK implemented endpoints: %d", len(implemented))
	t.Logf("Missing endpoints: %d", len(missing))

	for _, ep := range missing {
		t.Logf("  missing: %s", ep)
	}

	assert.Empty(t, missing, fmt.Sprintf("%d spec endpoints not implemented in SDK", len(missing)))
}
