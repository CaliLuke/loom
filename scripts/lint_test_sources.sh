#!/usr/bin/env bash
# Lints the hand-written Go sources that the root-module `./...` lint skips:
#
#   - testdata packages, which Go package patterns never match;
#   - nested modules other than the integration fixtures, such as
#     http/integration_tests and jsonrpc/integration_tests.
#
# The integration fixture modules under */integration_tests/fixtures/ are
# excluded: `make generated-code-quality` lints their regenerated gen/ trees.
# Generated gen/ trees are excluded everywhere for the same reason.
#
# Each target is checked with the root .golangci.yml and the same staticcheck
# checks as the root-module lint.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLANGCI_LINT="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"
STATICCHECK="${STATICCHECK:-$(go env GOPATH)/bin/staticcheck}"
STATICCHECK_CHECKS="${STATICCHECK_CHECKS:-all,-S*,-ST*,-QF*}"
CONFIG="$ROOT/.golangci.yml"

for tool in "$GOLANGCI_LINT" "$STATICCHECK"; do
  if [ ! -x "$tool" ]; then
    echo "missing lint tool: $tool" >&2
    echo "run: make depend" >&2
    exit 1
  fi
done

cd "$ROOT"

# module_dirs prints every module directory except the integration fixtures.
module_dirs() {
  find . -path ./.git -prune -o -type f -name go.mod -print |
    sed 's|/go\.mod$||' |
    grep -Ev '/integration_tests/fixtures(/|$)' |
    LC_ALL=C sort
}

# testdata_dirs MODULE prints the testdata package directories of MODULE,
# relative to MODULE. It skips gen/ trees and every module nested below MODULE.
testdata_dirs() {
  local module="$1"
  local prune=(-path ./.git -o -type d -name gen)
  local nested
  while IFS= read -r nested; do
    prune+=(-o -path "$nested")
  done < <(cd "$module" && find . -mindepth 2 -type f -name go.mod ! -path './.git/*' | sed 's|/go\.mod$||')
  (cd "$module" && find . \( "${prune[@]}" \) -prune -o -type f -name '*.go' -path '*/testdata/*' -print) |
    sed 's|/[^/]*$||' |
    LC_ALL=C sort -u
}

status=0
linted_testdata=0
linted_modules=0

while IFS= read -r module; do
  targets=()
  if [ "$module" != "." ]; then
    targets+=("./...")
    linted_modules=$((linted_modules + 1))
  fi
  while IFS= read -r dir; do
    [ -n "$dir" ] || continue
    targets+=("$dir")
    linted_testdata=$((linted_testdata + 1))
  done < <(testdata_dirs "$module")
  if [ "${#targets[@]}" -eq 0 ]; then
    continue
  fi

  echo "==> test sources in module ${module#./}"
  if ! (cd "$module" && "$STATICCHECK" -checks="$STATICCHECK_CHECKS" "${targets[@]}"); then
    status=1
  fi
  if ! (cd "$module" && "$GOLANGCI_LINT" run --config "$CONFIG" "${targets[@]}"); then
    status=1
  fi
done < <(module_dirs)

# Guard against discovery silently matching nothing.
if [ "$linted_modules" -eq 0 ] || [ "$linted_testdata" -eq 0 ]; then
  echo "test source lint discovered $linted_modules nested modules and $linted_testdata testdata packages; expected both" >&2
  exit 1
fi

exit "$status"
