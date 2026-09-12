#!/usr/bin/env bash
# Re-vendor the Kalshi API spec snapshot in specs/.
#
# Generators and drift tests read the vendored copies, so refreshing them is a
# deliberate, reviewable act rather than a silent dependency on whatever
# docs.kalshi.com served at build time.
#
#   ./tools/vendor_spec.sh          # fetch live, rewrite specs/, report the diff
#   ./tools/vendor_spec.sh --check  # exit 1 if live differs from vendored (CI)
set -euo pipefail

OPENAPI_URL="https://docs.kalshi.com/openapi.yaml"
ASYNCAPI_URL="https://docs.kalshi.com/asyncapi.yaml"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
specs_dir="$repo_root/specs"
check_only=false
[[ "${1:-}" == "--check" ]] && check_only=true

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fetch() {
  curl --fail --silent --show-error --location --max-time 60 -o "$2" "$1" \
    || { echo "error: could not fetch $1" >&2; exit 2; }
}

fetch "$OPENAPI_URL" "$tmp/openapi.yaml"
fetch "$ASYNCAPI_URL" "$tmp/asyncapi.yaml"

drifted=false
for name in openapi asyncapi; do
  live="$tmp/$name.yaml"
  vendored="$specs_dir/$name.yaml"
  if [[ ! -f "$vendored" ]] || ! cmp -s "$live" "$vendored"; then
    drifted=true
    echo "DRIFT: $name.yaml differs from vendored copy"
    if [[ -f "$vendored" ]]; then
      diff <(grep -c '' "$vendored") <(grep -c '' "$live") >/dev/null \
        || echo "  lines: $(grep -c '' "$vendored") -> $(grep -c '' "$live")"
    fi
  fi
done

if ! $drifted; then
  echo "specs/ is up to date with $OPENAPI_URL and $ASYNCAPI_URL"
  exit 0
fi

if $check_only; then
  echo
  echo "Vendored specs are behind. Run ./tools/vendor_spec.sh, then regenerate." >&2
  exit 1
fi

oa_version="$(awk '/^info:/{f=1} f&&/^  version:/{print $2; exit}' "$tmp/openapi.yaml")"
aa_info_version="$(awk '/^info:/{f=1} f&&/^  version:/{print $2; exit}' "$tmp/asyncapi.yaml")"
aa_version="$(awk '/^asyncapi:/{print $2; exit}' "$tmp/asyncapi.yaml")"

mkdir -p "$specs_dir"
cp "$tmp/openapi.yaml" "$specs_dir/openapi.yaml"
cp "$tmp/asyncapi.yaml" "$specs_dir/asyncapi.yaml"

cat > "$specs_dir/SNAPSHOT.json" <<EOF
{
  "_comment": "Pinned Kalshi API spec snapshot. Generators and drift tests read these vendored files so builds are reproducible and every release is traceable to an exact spec. A scheduled job compares these against live and opens a PR when they diverge. Do not hand-edit the YAML; re-vendor with tools/vendor_spec.sh.",
  "fetched_at": "$(date -u +%Y-%m-%d)",
  "openapi": {
    "url": "$OPENAPI_URL",
    "version": "$oa_version",
    "sha256": "$(shasum -a 256 "$specs_dir/openapi.yaml" | cut -d' ' -f1)"
  },
  "asyncapi": {
    "url": "$ASYNCAPI_URL",
    "info_version": "$aa_info_version",
    "asyncapi_version": "$aa_version",
    "sha256": "$(shasum -a 256 "$specs_dir/asyncapi.yaml" | cut -d' ' -f1)"
  }
}
EOF

echo
echo "Re-vendored. OpenAPI $oa_version, AsyncAPI info $aa_info_version."
echo "Next: regenerate types and run the drift tests."
