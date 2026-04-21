#!/usr/bin/env bash
# Lint for manual package-qualified type-reference assembly inside codegen.
#
# Codegen must build qualified Go type references (e.g. `myservice.UserType`)
# via NameScope helpers — GoFullTypeRef / GoFullTypeName — so that imports
# are tracked and qualifiers stay in sync with the scope. Hand-building a
# qualified reference through fmt.Sprintf("%s.%s", ...) or string concat
# bypasses that machinery and produces subtly wrong references when the
# referenced type lives in a different package than expected.
#
# This lint rejects the highest-signal pattern: fmt.Sprintf with a
# "%s.%s" or "%s.%s%s" template, which is the hallmark of trying to
# assemble "<package>.<TypeName>" manually. Lower-signal concat patterns
# ("*" + ident, etc.) are intentionally not flagged here because they have
# too many legitimate uses (pointer derefs, field expressions, initializer
# prefixes) and would require blanket nolint annotations.
#
# To opt out of a specific site (rare), annotate with:
#   // nolint: namescope

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

TARGETS=(
  "${ROOT_DIR}/codegen"
  "${ROOT_DIR}/http/codegen"
  "${ROOT_DIR}/grpc/codegen"
  "${ROOT_DIR}/jsonrpc/codegen"
)

# Hand-assembled "<package>.<Type>" references. Two variants caught:
#   fmt.Sprintf("%s.%s", pkg, TypeName)
#   fmt.Sprintf("%s.%s%s", pkg, TypeName, suffix)
PATTERN='fmt\.Sprintf\(\s*"%s\.%s'

exit_status=0

for dir in "${TARGETS[@]}"; do
  if [[ ! -d "${dir}" ]]; then
    continue
  fi
  while IFS= read -r -d '' file; do
    case "${file}" in
      */testdata/*|*/testing.go|*_test.go|*/gen/*) continue ;;
    esac
    matches="$(grep -nHE "${PATTERN}" "${file}" 2>/dev/null | grep -v 'nolint: namescope' || true)"
    if [[ -n "${matches}" ]]; then
      echo "name-scope lint: manual package-qualified type reference:" >&2
      echo "${matches}" >&2
      exit_status=1
    fi
  done < <(find "${dir}" -type f -name '*.go' -print0)
done

if (( exit_status != 0 )); then
  cat >&2 <<MSG

Route the reference through a NameScope helper instead:
  scope.GoFullTypeRef(att, pkg)
  scope.GoFullTypeName(att, pkg)

See CLAUDE.md "Codegen Implementation" rule. If the match really is not a
type reference, annotate the line with // nolint: namescope.
MSG
fi

exit "${exit_status}"
