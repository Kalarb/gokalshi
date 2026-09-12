//go:build spec_validation

package gokalshi

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// skippedPathPrefixes are spec paths excluded from the comparison. Empty: the
// SDK targets full spec coverage. An entry here must carry the reason it is not
// simply implemented.
var skippedPathPrefixes = []string{}

// knownExtraEndpoints are endpoints the SDK implements that the published spec
// does not list, each with the reason it is kept. Anything absent from the spec
// and absent from this map fails the test: Kalshi removes endpoints (and starts
// answering them with 410) faster than a coverage-only check can notice, so an
// unexplained extra is treated as dead code until proven otherwise.
var knownExtraEndpoints = map[string]string{}

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
	body := loadVendoredSpec(t, "openapi")

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

	// Find endpoints the SDK ships that the spec no longer lists. Endpoints
	// under a skipped prefix are excluded — the skip list means "not compared",
	// not "must not exist".
	var extra []string
	for ep := range implemented {
		if specEndpoints[ep] {
			continue
		}
		path := ep[strings.Index(ep, " ")+1:]
		if shouldSkipPath(path) {
			continue
		}
		if _, known := knownExtraEndpoints[ep]; known {
			continue
		}
		extra = append(extra, ep)
	}
	sort.Strings(extra)

	t.Logf("OpenAPI spec endpoints: %d (after skipping %v)", len(specEndpoints), skippedPathPrefixes)
	t.Logf("SDK implemented endpoints: %d", len(implemented))
	t.Logf("Missing endpoints: %d", len(missing))
	t.Logf("Extra endpoints (not in spec): %d", len(extra))

	for _, ep := range missing {
		t.Logf("  missing: %s", ep)
	}
	for _, ep := range extra {
		t.Logf("  extra (not in spec): %s", ep)
	}

	assert.Empty(t, missing, fmt.Sprintf("%d spec endpoints not implemented in SDK", len(missing)))
	assert.Empty(t, extra, fmt.Sprintf(
		"%d SDK endpoints are absent from the spec — remove them, or add them to "+
			"knownExtraEndpoints with a reason", len(extra)))
}
