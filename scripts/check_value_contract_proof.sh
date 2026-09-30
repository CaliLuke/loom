#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
project="$root/expr/lean/value_projection"
cd "$project"

if ! command -v lake >/dev/null 2>&1; then
  echo 'missing lake; install the toolchain pinned in expr/lean/value_projection/lean-toolchain' >&2
  exit 1
fi

toolchain=$(cat lean-toolchain)
case "$toolchain" in
  leanprover/lean4:v*) expected_version=${toolchain#leanprover/lean4:v} ;;
  *) echo "invalid pinned Lean toolchain: $toolchain" >&2; exit 1 ;;
esac

proof_tmp=$(mktemp -d "${TMPDIR:-/tmp}/loom-value-proof.XXXXXX")
trap 'rm -rf "$proof_tmp"' EXIT

run_check() {
  local label=$1
  local output=$2
  shift 2
  if ! "$@" >"$output" 2>&1; then
    cat "$output" >&2
    echo "$label failed" >&2
    return 1
  fi
  cat "$output"
  if LC_ALL=C grep -Eq '(^|[^[:alnum:]_])warning:|⚠' "$output"; then
    echo "$label emitted a warning" >&2
    return 1
  fi
}

run_check 'toolchain check' "$proof_tmp/version" lake env lean --version
if ! grep -Fq "Lean (version $expected_version," "$proof_tmp/version"; then
  echo "expected Lean $expected_version from $toolchain" >&2
  exit 1
fi

run_check 'proof build' "$proof_tmp/build" lake build

run_negative() {
  local source=$1
  local output=$2
  shift 2
  if lake env lean -DwarningAsError=true "$source" >"$output" 2>&1; then
    cat "$output" >&2
    echo "negative proof control unexpectedly succeeded: $source" >&2
    return 1
  fi
  local expected
  for expected in "$@"; do
    if ! grep -Fq "$expected" "$output"; then
      cat "$output" >&2
      echo "negative proof control failed for an unexpected reason: $source" >&2
      return 1
    fi
  done
}

run_negative NegativeNearestPattern.lean "$proof_tmp/negative-pattern" \
  'proved that the proposition' 'is false'
run_negative NegativeNearestFormat.lean "$proof_tmp/negative-format" \
  'proved that the proposition' 'is false'
run_negative NegativeNumericOverwrite.lean "$proof_tmp/negative-numeric-overwrite" \
  'error: unsolved goals' '⊢ False'
run_negative NegativeNumericClosedTie.lean "$proof_tmp/negative-numeric-closed-tie" \
  'proved that the proposition' 'is false'
run_negative NegativeEnumOverride.lean "$proof_tmp/negative-enum-override" \
  'proved that the proposition' 'is false'
run_check 'transitive axiom audit' "$proof_tmp/audit" \
  lake env lean -DwarningAsError=true ValueContract/AxiomAudit.lean

if ! grep -Eq '^VALUE_CONTRACT_AUDIT_OK [1-9][0-9]*$' "$proof_tmp/audit"; then
  echo 'missing audit completion for the required theorem manifest' >&2
  exit 1
fi

count=0
while IFS= read -r theorem || [[ -n "$theorem" ]]; do
  if [[ -z "$theorem" ]]; then
    continue
  fi
  if ! grep -Fq "VALUE_CONTRACT_THEOREM $theorem axioms=" "$proof_tmp/audit"; then
    echo "missing theorem report: $theorem" >&2
    exit 1
  fi
  count=$((count + 1))
done < required-theorems.txt

if [[ "$count" -eq 0 ]] || ! grep -Fxq "VALUE_CONTRACT_AUDIT_OK $count" "$proof_tmp/audit"; then
  echo 'missing audit completion for the required theorem manifest' >&2
  exit 1
fi

run_check 'fresh kernel replay' "$proof_tmp/kernel" \
  lake env leanchecker --fresh ValueContract.Proofs
echo "value contract proof gate passed ($count required theorems)"
