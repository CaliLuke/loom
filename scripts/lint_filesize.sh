#!/usr/bin/env bash
# Go file-size lint.
#
# Enforces a hard upper bound on Go source file length — files over
# MAX_GO_FILE_LINES fail the build. Also prints a non-blocking warning for
# files over WARN_GO_FILE_LINES so contributors have advance notice before a
# file hits the hard cap.
#
# Tune via env vars (useful for a ratcheted migration):
#   MAX_GO_FILE_LINES=1000   hard block; files above this fail the script.
#   WARN_GO_FILE_LINES=700   soft warning; reported but does not fail.
#
# Test files (*_test.go) are exempt from the hard cap because table-driven
# case blocks legitimately push them longer than production files. They still
# participate in the warning tier.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAX_GO_FILE_LINES="${MAX_GO_FILE_LINES:-700}"
# Warn threshold defaults to 100 below the hard cap so contributors get a
# nudge before hitting it.
WARN_GO_FILE_LINES="${WARN_GO_FILE_LINES:-600}"

validate_int() {
  local name="$1" value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]]; then
    echo "${name} must be an integer, got: ${value}" >&2
    exit 2
  fi
}
validate_int MAX_GO_FILE_LINES "${MAX_GO_FILE_LINES}"
validate_int WARN_GO_FILE_LINES "${WARN_GO_FILE_LINES}"

status=0
warn_count=0

while IFS= read -r -d '' file; do
  lines="$(wc -l < "${file}")"
  rel="${file#${ROOT_DIR}/}"
  is_test=0
  case "${file}" in
    *_test.go) is_test=1 ;;
  esac

  if (( lines > MAX_GO_FILE_LINES )) && (( is_test == 0 )); then
    bytes="$(wc -c < "${file}")"
    printf 'oversize Go file: %s (%s lines, %s bytes; hard limit %s)\n' \
      "${rel}" "${lines}" "${bytes}" "${MAX_GO_FILE_LINES}" >&2
    status=1
    continue
  fi

  if (( lines > WARN_GO_FILE_LINES )); then
    suffix=""
    if (( is_test == 1 )); then
      suffix=" [test]"
    fi
    printf 'WARNING: large Go file: %s (%s lines; warn threshold %s)%s\n' \
      "${rel}" "${lines}" "${WARN_GO_FILE_LINES}" "${suffix}" >&2
    warn_count=$((warn_count + 1))
  fi
done < <(find "${ROOT_DIR}" \
  -type f \
  -name '*.go' \
  -not -path "${ROOT_DIR}/.git/*" \
  -not -path '*/gen/*' \
  -not -path "*/testdata/*" \
  -print0)

if (( warn_count > 0 )); then
  echo "file-size lint: ${warn_count} file(s) above warn threshold (${WARN_GO_FILE_LINES} lines); consider splitting before they hit the hard cap." >&2
fi

if (( status != 0 )); then
  echo "file-size lint failed: split oversized Go files or raise MAX_GO_FILE_LINES intentionally" >&2
fi

exit "${status}"
