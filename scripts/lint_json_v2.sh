#!/usr/bin/env bash
set -euo pipefail

# Selvedge owns durable input encoding. Its exact codec bridge is the sole
# native JSON exception; other runtime, generator and test files remain v2.
NATIVE_CODEC=execution/selvedge/workflow_encoding.go

matches="$({
  git grep -n -E '"encoding/json"' -- '*.go' '*.golden' '*.tmpl' ":(exclude)$NATIVE_CODEC" || true
  git grep -n -E 'encoding/json[^/]?`' -- '*.go' '*.md' || true
  git grep -n -E 'json\.(RawMessage|Number|NewEncoder|NewDecoder|MarshalIndent|Valid)([^[:alnum:]_]|$)' -- '*.go' '*.golden' '*.tmpl' ":(exclude)$NATIVE_CODEC" || true
} | sort -u)"

if [[ -n "$matches" ]]; then
  printf '%s\n' "$matches" >&2
  echo "Loom-owned JSON must use encoding/json/v2 and encoding/json/jsontext." >&2
  exit 1
fi

go run ./scripts/jsonv2check
