#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAKEFILE="$ROOT/Makefile"
WORKFLOW="$ROOT/.github/workflows/test.yml"
CHECK_SCRIPT="$ROOT/check.sh"
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
git -C "$ROOT" tag -a "$release_version" -m "CI contract test"
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

echo "CI contract lint passed"
