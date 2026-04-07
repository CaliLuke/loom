#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAX_GO_FILE_LINES="${MAX_GO_FILE_LINES:-1000}"

if ! [[ "${MAX_GO_FILE_LINES}" =~ ^[0-9]+$ ]]; then
  echo "MAX_GO_FILE_LINES must be an integer, got: ${MAX_GO_FILE_LINES}" >&2
  exit 2
fi

status=0

while IFS= read -r -d '' file; do
  lines="$(wc -l < "${file}")"
  if (( lines > MAX_GO_FILE_LINES )); then
    bytes="$(wc -c < "${file}")"
    rel="${file#${ROOT_DIR}/}"
    printf 'oversize Go file: %s (%s lines, %s bytes; limit %s lines)\n' \
      "${rel}" "${lines}" "${bytes}" "${MAX_GO_FILE_LINES}" >&2
    status=1
  fi
done < <(find "${ROOT_DIR}" \
  -type f \
  -name '*.go' \
  -not -path "${ROOT_DIR}/.git/*" \
  -not -path "${ROOT_DIR}/gen/*" \
  -print0)

if (( status != 0 )); then
  echo "file-size lint failed: split oversized Go files or raise MAX_GO_FILE_LINES intentionally" >&2
fi

exit "${status}"
