//go:build spec_validation

package gokalshi

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// specSnapshot mirrors specs/SNAPSHOT.json.
type specSnapshot struct {
	FetchedAt string `json:"fetched_at"`
	OpenAPI   struct {
		Version string `json:"version"`
	} `json:"openapi"`
	AsyncAPI struct {
		InfoVersion string `json:"info_version"`
	} `json:"asyncapi"`
}

// loadVendoredSpec reads a pinned spec from specs/.
//
// Drift tests read the vendored snapshot rather than docs.kalshi.com so they
// are deterministic and offline-capable: they answer "does the SDK match the
// spec we generated from?". A separate scheduled job runs
// tools/vendor_spec.sh --check to answer "has Kalshi moved?".
func loadVendoredSpec(t *testing.T, kind string) []byte {
	t.Helper()

	body, err := os.ReadFile("specs/" + kind + ".yaml")
	require.NoError(t, err, "read vendored specs/%s.yaml — run ./tools/vendor_spec.sh", kind)

	snapBytes, err := os.ReadFile("specs/SNAPSHOT.json")
	require.NoError(t, err, "read specs/SNAPSHOT.json")

	var snap specSnapshot
	require.NoError(t, json.Unmarshal(snapBytes, &snap), "parse specs/SNAPSHOT.json")

	version := snap.OpenAPI.Version
	if kind == "asyncapi" {
		version = snap.AsyncAPI.InfoVersion
	}
	t.Logf("vendored %s spec version %s (fetched %s)", kind, version, snap.FetchedAt)

	return body
}
