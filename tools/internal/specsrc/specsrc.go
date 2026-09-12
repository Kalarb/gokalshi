// Package specsrc loads the vendored Kalshi API spec snapshot.
//
// Generators read the pinned copies in specs/ rather than fetching
// docs.kalshi.com at build time, so a regeneration is reproducible and every
// release is traceable to an exact spec version and hash. Refresh the snapshot
// with tools/vendor_spec.sh; CI compares it against live on a schedule.
package specsrc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Kind identifies which vendored spec to load.
type Kind string

const (
	OpenAPI  Kind = "openapi"
	AsyncAPI Kind = "asyncapi"
)

// Snapshot mirrors specs/SNAPSHOT.json.
type Snapshot struct {
	FetchedAt string `json:"fetched_at"`
	OpenAPI   struct {
		URL     string `json:"url"`
		Version string `json:"version"`
		SHA256  string `json:"sha256"`
	} `json:"openapi"`
	AsyncAPI struct {
		URL             string `json:"url"`
		InfoVersion     string `json:"info_version"`
		AsyncAPIVersion string `json:"asyncapi_version"`
		SHA256          string `json:"sha256"`
	} `json:"asyncapi"`
}

// Provenance returns the source URL, version and hash for the given kind,
// suitable for stamping into a generated file header.
func (s Snapshot) Provenance(kind Kind) (url, version, sha string) {
	if kind == AsyncAPI {
		return s.AsyncAPI.URL, s.AsyncAPI.InfoVersion, s.AsyncAPI.SHA256
	}
	return s.OpenAPI.URL, s.OpenAPI.Version, s.OpenAPI.SHA256
}

// RepoRoot walks up from the working directory to the module root.
func RepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("cannot find module root (go.mod) above working directory")
}

// Load reads the vendored spec of the given kind along with its snapshot
// metadata.
func Load(kind Kind) ([]byte, Snapshot, error) {
	var snap Snapshot

	root, err := RepoRoot()
	if err != nil {
		return nil, snap, err
	}
	specPath := filepath.Join(root, "specs", string(kind)+".yaml")

	body, err := os.ReadFile(specPath)
	if err != nil {
		return nil, snap, fmt.Errorf("read vendored spec %s: %w (run ./tools/vendor_spec.sh)", specPath, err)
	}

	snapPath := filepath.Join(root, "specs", "SNAPSHOT.json")
	snapBytes, err := os.ReadFile(snapPath)
	if err != nil {
		return nil, snap, fmt.Errorf("read %s: %w (run ./tools/vendor_spec.sh)", snapPath, err)
	}
	if err := json.Unmarshal(snapBytes, &snap); err != nil {
		return nil, snap, fmt.Errorf("parse %s: %w", snapPath, err)
	}
	return body, snap, nil
}
