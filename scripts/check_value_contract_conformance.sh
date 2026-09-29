#!/usr/bin/env bash
# Make owns the prerequisite proof gate. This runner builds and checks the same
# pinned reference project, then verifies both executed tests and owner coverage.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT="$ROOT/expr/lean/value_projection"
fail() {
  echo "value contract conformance: $*" >&2
  exit 1
}
command -v lake >/dev/null 2>&1 || fail "missing lake (install the pinned Lean toolchain)"
command -v go >/dev/null 2>&1 || fail "missing go"
TASK_TMP="$(mktemp -d "${TMPDIR:-/tmp}/loom-value-conformance.XXXXXX")"
trap 'rm -rf "$TASK_TMP"' EXIT

cd "$PROJECT"
TOOLCHAIN="$(cat lean-toolchain)"
VERSION="${TOOLCHAIN#leanprover/lean4:v}"
[[ "$VERSION" != "$TOOLCHAIN" ]] || fail "unsupported Lean toolchain declaration"
ACTUAL="$(lake env lean --version)"
[[ "$ACTUAL" == *"version $VERSION,"* ]] || fail "expected Lean $VERSION, got $ACTUAL"
if ! lake build value_contract_reference >"$TASK_TMP/build.log" 2>&1; then
  cat "$TASK_TMP/build.log"
  fail "reference build failed"
fi
cat "$TASK_TMP/build.log"
if grep -Ei '(^|[[:space:]])warning:' "$TASK_TMP/build.log"; then
  fail "reference build emitted a warning"
fi
REFERENCE="$PROJECT/.lake/build/bin/value_contract_reference"
[[ -x "$REFERENCE" ]] || fail "missing built reference executable"

cd "$ROOT"
TEST_STATUS=0
LOOM_VALUE_CONFORMANCE=1 LOOM_LEAN_REFERENCE="$REFERENCE" \
  LOOM_VALUE_CONFORMANCE_REPORT="$TASK_TMP/conformance-results.json" \
  go test -json ./internal/valuecontract -run '^TestLeanConformance$' -count=1 -timeout=10m \
  >"$TASK_TMP/events.jsonl" 2>&1 || TEST_STATUS=$?
cat "$TASK_TMP/events.jsonl"
go run ./scripts/valuecontractcheck -events "$TASK_TMP/events.jsonl" -report "$TASK_TMP/conformance-results.json"
[[ "$TEST_STATUS" == 0 ]] || fail "Go conformance exited $TEST_STATUS"
