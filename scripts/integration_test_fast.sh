#!/usr/bin/env bash

set -euo pipefail

service="${SERVICE:-${1:-ticktock}}"
run_pattern="${RUN:-${2:-.}}"
repo_root="${3:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
gobin_dir="${GOBIN_DIR:-$(go env GOPATH)/bin}"

if [[ ! "$service" =~ ^[[:alnum:]][[:alnum:]_.-]*$ ]]; then
  echo "invalid integration fixture service: $service" >&2
  exit 2
fi

fixtures=(
  "$repo_root/jsonrpc/integration_tests/fixtures/$service"
  "$repo_root/http/integration_tests/fixtures/$service"
)

for fixture in "${fixtures[@]}"; do
  if [[ ! -f "$fixture/go.mod" ]]; then
    echo "integration fixture module not found: $fixture" >&2
    exit 2
  fi
done

cleanup() {
  local status=$?
  trap - EXIT
  for fixture in "${fixtures[@]}"; do
    find "$fixture" -type f -name 'server-*.log' -delete || status=$?
    find "$fixture" -type d -name 'loom[0-9]*' -prune -exec rm -rf -- {} + || status=$?
  done
  exit "$status"
}
trap cleanup EXIT

for fixture in "${fixtures[@]}"; do
  (
    cd "$fixture"
    PATH="$gobin_dir:$PATH" go test -count=1 -timeout 5m -run "$run_pattern" ./...
  )
done
