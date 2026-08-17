#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_BIN="$(command -v go)"
MAKE_BIN="$(command -v make)"
LOOM_BINARY="$(go env GOPATH)/bin/loom"
MAKEFILE="$ROOT/Makefile"
WORKFLOW="$ROOT/.github/workflows/test.yml"
CHECK_SCRIPT="$ROOT/check.sh"
FAST_INTEGRATION_SCRIPT="$ROOT/scripts/integration_test_fast.sh"
PRE_COMMIT="$ROOT/.githooks/pre-commit"
PRE_PUSH="$ROOT/.githooks/pre-push"

fail() {
  echo "CI contract lint: $*" >&2
  exit 1
}

TMP_BASE="$(mktemp -d "${TMPDIR:-/tmp}/loom-ci-contract.XXXXXX")"
trap 'rm -rf "$TMP_BASE"' EXIT

MAKE_DATABASE="$TMP_BASE/make.db"
make -npRr -f "$MAKEFILE" all >"$MAKE_DATABASE" || fail "could not inspect Make targets"

target_prerequisites() {
  local target="$1"
  awk -v target="$target" '
    $1 == target ":" {
      for (i = 2; i <= NF; i++) {
        printf "%s%s", (i == 2 ? "" : " "), $i
      }
      print ""
      found = 1
      exit
    }
    END {
      if (!found) {
        exit 1
      }
    }
  ' "$MAKE_DATABASE"
}

assert_prerequisites() {
  local target="$1"
  local expected="$2"
  local actual
  actual="$(target_prerequisites "$target")" || fail "missing Make target $target"
  if [[ "$actual" != "$expected" ]]; then
    fail "Make target $target prerequisites are [$actual], want [$expected]"
  fi
}

assert_prerequisites all "lint test integration-test"
assert_prerequisites ci "depend all"
assert_prerequisites ci-local "all test-race openapi-contract generated-code-quality"

fast_recipe="$(make --no-print-directory -C "$ROOT" -n integration-test-fast SERVICE=ticktock RUN='^TestFast$$')"
expected_fast_recipe="bash ./scripts/integration_test_fast.sh"
if ! grep -Fqx "$expected_fast_recipe" <<<"$fast_recipe"; then
  fail "integration-test-fast does not invoke the nested fixture runner"
fi

if grep -Eq '^[[:space:]]*run:[[:space:]]*[|>][+-]?[[:space:]]*(#.*)?$' "$WORKFLOW"; then
  fail "workflow multiline run blocks are unsupported because Make targets could escape CI contract extraction"
fi

workflow_targets="$({
  sed -nE 's/^[[:space:]]*run:[[:space:]]*make[[:space:]]+([^[:space:]#]+).*$/\1/p' "$WORKFLOW"
} | LC_ALL=C sort -u)"
expected_workflow_targets="$(printf '%s\n' \
  ci \
  depend \
  generated-code-quality \
  openapi-contract \
  test-race \
  | LC_ALL=C sort)"
if [[ "$workflow_targets" != "$expected_workflow_targets" ]]; then
  fail "workflow Make targets are [$workflow_targets], want [$expected_workflow_targets]"
fi

STUB_BIN="$TMP_BASE/bin"
MAKE_LOG="$TMP_BASE/make.log"
mkdir -p "$STUB_BIN"
cat >"$STUB_BIN/make" <<'EOF'
#!/bin/sh
printf '%s|%s|%s|%s\n' "${LOOM_DIR-}" "${GIT_DIR-}" "${GIT_WORK_TREE-}" "$*" >>"$CI_CONTRACT_MAKE_LOG"
EOF
chmod 0755 "$STUB_BIN/make"

run_and_assert_make() {
  local expected="$1"
  shift
  : >"$MAKE_LOG"
  CI_CONTRACT_MAKE_LOG="$MAKE_LOG" PATH="$STUB_BIN:$PATH" LOOM_DIR= "$@" >/dev/null
  local actual
  actual="$(<"$MAKE_LOG")"
  if [[ "$actual" != "$expected" ]]; then
    fail "$* invoked Make as [$actual], want [$expected]"
  fi
}

run_and_assert_make "|||ci-local" "$CHECK_SCRIPT" --full
run_and_assert_make "|||lint test" "$CHECK_SCRIPT"

repo_root="$(git -C "$ROOT" rev-parse --show-toplevel)"
git_dir="$(git -C "$ROOT" rev-parse --absolute-git-dir)"
run_and_assert_make "|||lint" env GIT_DIR="$git_dir" GIT_WORK_TREE="$repo_root" "$PRE_COMMIT"

update_oid="1111111111111111111111111111111111111111"
remote_oid="2222222222222222222222222222222222222222"
delete_oid="0000000000000000000000000000000000000000"

run_hook_and_assert_make() {
  local input="$1"
  local expected="$2"
  local release_version="${3-}"
  : >"$MAKE_LOG"
  local git_dir
  git_dir="$(git -C "$ROOT" rev-parse --absolute-git-dir)"
  printf '%b' "$input" | CI_CONTRACT_MAKE_LOG="$MAKE_LOG" PATH="$STUB_BIN:$PATH" LOOM_DIR= \
    GIT_DIR="$git_dir" GIT_WORK_TREE="$repo_root" LOOM_RELEASE_VERSION="$release_version" \
    "$PRE_PUSH" origin https://github.com/CaliLuke/loom.git >/dev/null
  local actual
  actual="$(<"$MAKE_LOG")"
  if [[ "$actual" != "$expected" ]]; then
    fail "pre-push invoked Make as [$actual], want [$expected]"
  fi
}

run_hook_and_assert_make \
  "refs/heads/main $update_oid refs/heads/main $remote_oid\n" \
  "$repo_root|||ci-local"
run_hook_and_assert_make \
  "refs/heads/feature $update_oid refs/heads/feature $remote_oid\n" \
  "|||lint test"
run_hook_and_assert_make \
  "(delete) $delete_oid refs/heads/main $remote_oid\n" \
  "|||lint test"
run_hook_and_assert_make "" "|||lint test"

release_version="v9.8.7-alpha.6"
lightweight_version="v9.8.7-alpha.5"
release_head="$(git -C "$ROOT" rev-parse HEAD)"
release_tag="refs/tags/$release_version"
git -C "$ROOT" -c user.name="Loom CI Contract" -c user.email="loom-ci-contract@example.com" \
  tag -a "$release_version" -m "CI contract test"
git -C "$ROOT" tag "$lightweight_version"
release_tag_oid="$(git -C "$ROOT" rev-parse "$release_tag")"
lightweight_tag="refs/tags/$lightweight_version"
lightweight_tag_oid="$(git -C "$ROOT" rev-parse "$lightweight_tag")"
trap 'git -C "$ROOT" tag -d "$release_version" "$lightweight_version" >/dev/null 2>&1 || true; rm -rf "$TMP_BASE"' EXIT
run_hook_and_assert_make \
  "HEAD $release_head refs/heads/main $remote_oid\n$release_tag $release_tag_oid $release_tag $delete_oid\n" \
  "" \
  "$release_version"

git_dir="$(git -C "$ROOT" rev-parse --absolute-git-dir)"
if printf '%b' "HEAD $release_head refs/heads/main $remote_oid\n" | \
  CI_CONTRACT_MAKE_LOG="$MAKE_LOG" PATH="$STUB_BIN:$PATH" LOOM_DIR= \
  GIT_DIR="$git_dir" GIT_WORK_TREE="$repo_root" LOOM_RELEASE_VERSION="$release_version" \
  "$PRE_PUSH" origin https://github.com/CaliLuke/loom.git >/dev/null 2>&1; then
  fail "pre-push accepted an incomplete Loom release publication"
fi

if printf '%b' "HEAD $release_head refs/heads/main $remote_oid\n$lightweight_tag $lightweight_tag_oid $lightweight_tag $delete_oid\n" | \
  CI_CONTRACT_MAKE_LOG="$MAKE_LOG" PATH="$STUB_BIN:$PATH" LOOM_DIR= \
  GIT_DIR="$git_dir" GIT_WORK_TREE="$repo_root" LOOM_RELEASE_VERSION="$lightweight_version" \
  "$PRE_PUSH" origin https://github.com/CaliLuke/loom.git >/dev/null 2>&1; then
  fail "pre-push accepted a lightweight Loom release tag"
fi

FAST_ROOT="$TMP_BASE/integration-fast"
FAST_LOG="$TMP_BASE/integration-fast.log"
mkdir -p "$FAST_ROOT"
FAST_ROOT="$(cd "$FAST_ROOT" && pwd)"
for transport in jsonrpc http; do
  fixture="$FAST_ROOT/$transport/integration_tests/fixtures/ticktock"
  mkdir -p "$fixture/loom123"
  : >"$fixture/go.mod"
  : >"$fixture/server-existing.log"
  : >"$fixture/loom123/artifact"
done
cat >"$STUB_BIN/go" <<'EOF'
#!/bin/sh
if [ -z "${FAST_INTEGRATION_LOG-}" ]; then
  exec "$REAL_GO_BIN" "$@"
fi
printf '%s|%s\n' "$PWD" "$*" >>"$FAST_INTEGRATION_LOG"
: >server-new.log
mkdir -p loom456
: >loom456/artifact
if [ "${FAST_INTEGRATION_FAIL_DIR-}" = "$PWD" ]; then
  exit 9
fi
EOF
chmod 0755 "$STUB_BIN/go"

: >"$FAST_LOG"
FAST_INTEGRATION_LOG="$FAST_LOG" GOBIN_DIR="$STUB_BIN" \
  SERVICE= RUN= bash "$FAST_INTEGRATION_SCRIPT" ticktock '^TestFast$' "$FAST_ROOT"
expected_fast_log="$(printf '%s\n%s' \
  "$FAST_ROOT/jsonrpc/integration_tests/fixtures/ticktock|test -count=1 -timeout 5m -run ^TestFast$ ./..." \
  "$FAST_ROOT/http/integration_tests/fixtures/ticktock|test -count=1 -timeout 5m -run ^TestFast$ ./...")"
actual_fast_log="$(<"$FAST_LOG")"
if [[ "$actual_fast_log" != "$expected_fast_log" ]]; then
  fail "integration-test-fast invoked Go as [$actual_fast_log], want [$expected_fast_log]"
fi
if find "$FAST_ROOT" \( -type f -name 'server-*.log' -o -type d -name 'loom[0-9]*' \) -print -quit | grep -q .; then
  fail "integration-test-fast left server logs or loom temp directories"
fi

if FAST_INTEGRATION_LOG="$FAST_LOG" GOBIN_DIR="$STUB_BIN" \
  FAST_INTEGRATION_FAIL_DIR="$FAST_ROOT/http/integration_tests/fixtures/ticktock" \
  SERVICE= RUN= bash "$FAST_INTEGRATION_SCRIPT" ticktock '^TestFailure$' "$FAST_ROOT"; then
  fail "integration-test-fast ignored a fixture test failure"
fi
if find "$FAST_ROOT" \( -type f -name 'server-*.log' -o -type d -name 'loom[0-9]*' \) -print -quit | grep -q .; then
  fail "failed integration-test-fast left server logs or loom temp directories"
fi
if SERVICE= RUN= bash "$FAST_INTEGRATION_SCRIPT" '../ticktock' . "$FAST_ROOT" >/dev/null 2>&1; then
  fail "integration-test-fast accepted an unsafe fixture service"
fi

cat >"$STUB_BIN/bash" <<'EOF'
#!/bin/sh
printf '%s|%s|%s\n' "$*" "$SERVICE" "$RUN" >"$FAST_MAKE_LOG"
EOF
chmod 0755 "$STUB_BIN/bash"
unsafe_service="tick'tock; echo injected"
quoted_run="Test user's value; echo not-run"
FAST_MAKE_LOG="$FAST_LOG" REAL_GO_BIN="$GO_BIN" PATH="$STUB_BIN:$PATH" \
  "$MAKE_BIN" --no-print-directory -C "$ROOT" -o "$LOOM_BINARY" \
  integration-test-fast SERVICE="$unsafe_service" RUN="$quoted_run" >/dev/null
expected_make_fast="./scripts/integration_test_fast.sh|$unsafe_service|$quoted_run"
actual_make_fast="$(<"$FAST_LOG")"
if [[ "$actual_make_fast" != "$expected_make_fast" ]]; then
  fail "integration-test-fast changed or shell-parsed SERVICE/RUN at the Make boundary"
fi

echo "CI contract lint passed"
