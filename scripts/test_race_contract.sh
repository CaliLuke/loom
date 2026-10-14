#!/usr/bin/env bash
# Verify the race partition against the current root-module package inventory.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
TMP_BASE="$(mktemp -d "${TMPDIR:-/tmp}/loom-race-contract.XXXXXX")"
trap 'rm -rf "$TMP_BASE"' EXIT
export RACE_CONTRACT_INVENTORY="$TMP_BASE/inventory"
export RACE_CONTRACT_LOG="$TMP_BASE/invocation"
go list -race ./... >"$RACE_CONTRACT_INVENTORY"
recipe="$(make --no-print-directory -n test-race RACE_LANE=http LOOM_TEST_TIMEOUT=41m)"
grep -Fq 'bash ./scripts/test_race.sh "http" "41m"' <<<"$recipe"
default_recipe="$(make --no-print-directory -n test-race RACE_LANE=all LOOM_TEST_TIMEOUT=40m)"
grep -Fq 'bash ./scripts/test_race.sh "all" "40m"' <<<"$default_recipe"
grep -Fxq '        lane: [http, grpc, generator, remaining]' "$ROOT/.github/workflows/test.yml"
grep -Fxq '        run: make test-race RACE_LANE=${{ matrix.lane }}' "$ROOT/.github/workflows/test.yml"
cp "$RACE_CONTRACT_INVENTORY" "$TMP_BASE/original"
mkdir "$TMP_BASE/bin"
cat >"$TMP_BASE/bin/go" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$*" == 'list -race ./...' ]]; then
  cat "$RACE_CONTRACT_INVENTORY"
else
  printf '%s\n' "$@" >"$RACE_CONTRACT_LOG"
  exit "${RACE_CONTRACT_EXIT:-0}"
fi
STUB
chmod +x "$TMP_BASE/bin/go"
# A race-only package must survive discovery, not merely the final test flags.
# Use the real Go inventory in a minimal module so the assertion cannot mirror
# an incorrectly uninstrumented inventory stub.
REAL_RACE_GO="$(command -v go)"
export REAL_RACE_GO
race_root="$TMP_BASE/race-tags"
mkdir -p "$race_root/scripts" "$race_root/raceonly" "$TMP_BASE/race-bin"
printf 'module github.com/CaliLuke/loom\n\ngo 1.27.0\n' >"$race_root/go.mod"
printf 'package root\n' >"$race_root/root.go"
printf '//go:build race\n\npackage raceonly\n' >"$race_root/raceonly/race_test.go"
for owner in http/codegen grpc/codegen codegen/generator; do
  mkdir -p "$race_root/$owner"
  printf 'package owner\n' >"$race_root/$owner/owner.go"
done
cp "$ROOT/scripts/test_race.sh" "$race_root/scripts/test_race.sh"
cat >"$TMP_BASE/race-bin/go" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == list ]]; then
  exec "$REAL_RACE_GO" "$@"
fi
printf '%s\n' "$@" >"$RACE_CONTRACT_LOG"
STUB
chmod +x "$TMP_BASE/race-bin/go"
PATH="$TMP_BASE/race-bin:$PATH" bash "$race_root/scripts/test_race.sh" remaining 40m
grep -Fxq github.com/CaliLuke/loom/raceonly "$RACE_CONTRACT_LOG"

export PATH="$TMP_BASE/bin:$PATH"

fail() {
  echo "Race contract: $*" >&2
  exit 1
}

run_lane() {
  bash "$ROOT/scripts/test_race.sh" "$1" 40m
  head -n 6 "$RACE_CONTRACT_LOG" >"$TMP_BASE/flags"
  printf '%s\n' test -race -shuffle=on -count=1 -timeout 40m >"$TMP_BASE/expected-flags"
  cmp "$TMP_BASE/flags" "$TMP_BASE/expected-flags" || fail "changed race flags"
  tail -n +7 "$RACE_CONTRACT_LOG" >"$TMP_BASE/selected"
  [[ -s "$TMP_BASE/selected" ]] || fail "empty $1 lane"
}

run_lane all
[[ "$(cat "$TMP_BASE/selected")" == './...' ]] || fail "default is not the complete suite"
: >"$TMP_BASE/combined"
for lane in http grpc generator remaining; do
  run_lane "$lane"
  cat "$TMP_BASE/selected" >>"$TMP_BASE/combined"
  case "$lane" in
    http|grpc) expected="github.com/CaliLuke/loom/$lane/codegen" ;;
    generator) expected="github.com/CaliLuke/loom/codegen/generator" ;;
    remaining) continue ;;
  esac
  [[ "$(cat "$TMP_BASE/selected")" == "$expected" ]] || fail "incorrect dedicated lane"
done
LC_ALL=C sort "$TMP_BASE/combined" >"$TMP_BASE/actual"
LC_ALL=C sort "$TMP_BASE/original" >"$TMP_BASE/expected"
cmp "$TMP_BASE/actual" "$TMP_BASE/expected" || fail "partition is not disjoint and exhaustive"

if bash "$ROOT/scripts/test_race.sh" invalid 40m >"$TMP_BASE/error" 2>&1; then
  fail "unknown lane accepted"
fi
grep -q 'unknown race lane' "$TMP_BASE/error" || fail "unknown lane diagnostic missing"
: >"$RACE_CONTRACT_INVENTORY"
if bash "$ROOT/scripts/test_race.sh" http 40m >"$TMP_BASE/error" 2>&1; then
  fail "missing owner packages accepted"
fi
grep -q 'missing dedicated race package' "$TMP_BASE/error" || fail "missing owner diagnostic missing"
printf '%s\n' github.com/CaliLuke/loom/http/codegen github.com/CaliLuke/loom/grpc/codegen \
  github.com/CaliLuke/loom/codegen/generator >"$RACE_CONTRACT_INVENTORY"
if bash "$ROOT/scripts/test_race.sh" remaining 40m >"$TMP_BASE/error" 2>&1; then
  fail "empty remainder accepted"
fi
grep -q 'empty race lane' "$TMP_BASE/error" || fail "empty lane diagnostic missing"
cp "$TMP_BASE/original" "$RACE_CONTRACT_INVENTORY"
if RACE_CONTRACT_EXIT=7 bash "$ROOT/scripts/test_race.sh" http 40m; then
  fail "Go test failure ignored"
else
  status=$?
  [[ "$status" == 7 ]] || fail "Go test exit status changed"
fi
printf 'Race contract passed: %s packages across four exhaustive lanes\n' "$(wc -l <"$TMP_BASE/original" | tr -d ' ')"
