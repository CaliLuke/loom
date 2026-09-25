#!/usr/bin/env bash
# Formats the hand-written Go files that lint_gofmt.sh checks (see
# go_sources.sh). When goimports is installed it groups imports with
# github.com/CaliLuke/loom as the local prefix; gofmt -w then formats every
# discovered file. Both tools receive the file list, never a directory, so
# they cannot recurse into gen/ trees or the integration fixtures.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/go_sources.sh
source ./scripts/go_sources.sh

load_hand_written_go_files

if command -v goimports >/dev/null 2>&1; then
  goimports -w -local github.com/CaliLuke/loom "${GO_FILES[@]}"
else
  echo "goimports not on PATH; skipping import grouping (install: go install golang.org/x/tools/cmd/goimports@latest)"
fi

gofmt -w "${GO_FILES[@]}"
