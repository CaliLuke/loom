#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOOM_BIN="${LOOM_BIN:-$(go env GOPATH)/bin/loom}"
GOLANGCI_LINT="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"
STATICCHECK="${STATICCHECK:-$(go env GOPATH)/bin/staticcheck}"
STATICCHECK_CHECKS="${STATICCHECK_CHECKS:-all,-S*,-ST*,-QF*}"

if [ ! -x "$LOOM_BIN" ]; then
  echo "missing loom binary: $LOOM_BIN" >&2
  echo "run: make build-loom-cached" >&2
  exit 1
fi

if [ ! -x "$STATICCHECK" ]; then
  if command -v staticcheck >/dev/null 2>&1; then
    STATICCHECK="$(command -v staticcheck)"
  else
    echo "missing staticcheck: $STATICCHECK" >&2
    echo "run: make depend" >&2
    exit 1
  fi
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
GOLANGCI_LINT_CACHE="$TMP_BASE/.golangci-lint-cache"
export GOLANGCI_LINT_CACHE

fixtures=(
  "http-ticktock|http/integration_tests/fixtures/ticktock|example.com/http-ticktock|gen/http/clock/client,gen/http/clock/server,gen/http/openapi.json"
  "http-quality|http/integration_tests/fixtures/quality|example.com/http-quality|gen/http/accounts/client,gen/http/accounts/server,gen/http/openapi.json"
  "grpc-quality|grpc/integration_tests/fixtures/quality|example.com/grpc-quality|gen/grpc/accounts/client,gen/grpc/accounts/server,gen/grpc/accounts/pb/loomgen_grpc-quality_accounts.proto"
  "jsonrpc-ticktock|jsonrpc/integration_tests/fixtures/ticktock|example.com/ticktock|gen/jsonrpc/clock/client,gen/jsonrpc/clock/server"
  "jsonrpc-mixedtick|jsonrpc/integration_tests/fixtures/mixedtick|example.com/mixedtick|gen/jsonrpc/clock/client,gen/jsonrpc/clock/server"
)

run_fixture() {
  local label="$1"
  local fixture_path="$2"
  local module_path="$3"
  local required_gen_paths="$4"
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
    IFS=',' read -ra required_paths <<< "$required_gen_paths"
    for required_path in "${required_paths[@]}"; do
      if [ ! -e "$required_path" ]; then
        echo "missing required generated path for $label: $required_path" >&2
        exit 1
      fi
    done
    go mod tidy
    go test ./...
    go vet ./...
    "$STATICCHECK" -checks="$STATICCHECK_CHECKS" ./gen/...
    "$GOLANGCI_LINT" run \
      --no-config \
      --enable-only=errcheck,govet,ineffassign,unused,errorlint \
      --timeout=5m \
      ./gen/...
  )
}

for fixture in "${fixtures[@]}"; do
  IFS='|' read -r label path module_path required_gen_paths <<< "$fixture"
  run_fixture "$label" "$path" "$module_path" "$required_gen_paths"
done
