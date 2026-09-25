#!/usr/bin/env bash
# Fails when gofmt would reformat any hand-written Go file (see
# go_sources.sh for the covered and excluded trees). `make fmt` fixes them.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/go_sources.sh
source ./scripts/go_sources.sh

load_hand_written_go_files
unformatted="$(gofmt -l "${GO_FILES[@]}")"

if [ -n "$unformatted" ]; then
  echo "gofmt would reformat these files (run make fmt):" >&2
  printf '%s\n' "$unformatted" >&2
  exit 1
fi
