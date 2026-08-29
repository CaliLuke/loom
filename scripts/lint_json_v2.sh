#!/usr/bin/env bash
set -euo pipefail

matches="$({
  git grep -n -E '"encoding/json"' -- '*.go' '*.golden' '*.tmpl' || true
  git grep -n -E 'encoding/json[^/]?`' -- '*.go' '*.md' || true
  git grep -n -E 'json\.(RawMessage|Number|NewEncoder|NewDecoder|MarshalIndent|Valid)([^[:alnum:]_]|$)' -- '*.go' '*.golden' '*.tmpl' || true
} | sort -u)"

if [[ -n "$matches" ]]; then
  printf '%s\n' "$matches" >&2
  echo "Loom-owned JSON must use encoding/json/v2 and encoding/json/jsontext." >&2
  exit 1
fi
