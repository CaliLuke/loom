#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOOM_BIN="${LOOM_BIN:-$(go env GOPATH)/bin/loom}"
GOLANGCI_LINT="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"

if [ ! -x "$LOOM_BIN" ]; then
  echo "missing loom binary: $LOOM_BIN" >&2
  echo "run: make build-loom-cached" >&2
  exit 1
fi

if [ ! -x "$GOLANGCI_LINT" ]; then
  if command -v golangci-lint >/dev/null 2>&1; then
    GOLANGCI_LINT="$(command -v golangci-lint)"
  else
    echo "missing golangci-lint: $GOLANGCI_LINT" >&2
    echo "run: make depend" >&2
    exit 1
  fi
fi

TMP_BASE="$(mktemp -d "${TMPDIR:-/tmp}/loom-generated-quality.XXXXXX")"
trap 'rm -rf "$TMP_BASE"' EXIT

fixtures=(
  "http-ticktock|http/integration_tests/fixtures/ticktock|example.com/http-ticktock"
  "http-quality|http/integration_tests/fixtures/quality|example.com/http-quality"
  "jsonrpc-ticktock|jsonrpc/integration_tests/fixtures/ticktock|example.com/ticktock"
  "jsonrpc-mixedtick|jsonrpc/integration_tests/fixtures/mixedtick|example.com/mixedtick"
)

run_fixture() {
  local label="$1"
  local fixture_path="$2"
  local module_path="$3"
  local src="$ROOT/$fixture_path"
  local work="$TMP_BASE/$label"

  echo "==> $label"
  rsync -a \
    --exclude 'gen/' \
    --exclude 'loom[0-9]*/' \
    --exclude 'server-*.log' \
    "$src/" "$work/"

  (
    cd "$work"
    go mod edit -replace=github.com/CaliLuke/loom="$ROOT"
    "$LOOM_BIN" gen "$module_path/design"
    go mod tidy
    go test ./...
    go vet ./...
    "$GOLANGCI_LINT" run \
      --no-config \
      --enable-only=errcheck,staticcheck,govet,ineffassign,unused,errorlint \
      --timeout=5m \
      ./gen/...
  )
}

for fixture in "${fixtures[@]}"; do
  IFS='|' read -r label path module_path <<< "$fixture"
  run_fixture "$label" "$path" "$module_path"
done
