#!/usr/bin/env bash
# Quality-gate entry point for the Loom repo.
#
# This is a thin forwarder to the canonical `make` targets — the real gate
# logic lives in `Makefile`, `.golangci.yml`, `scripts/lint_filesize.sh`,
# and `scripts/lint_name_scope.sh`. The script exists so that external
# tooling (CI, pre-push hooks, agent harnesses) has a single stable entry
# point with the conventional name.
#
# Usage:
#   ./check.sh          # lint + unit tests
#   ./check.sh --fix    # auto-fix formatting/imports, then lint + unit tests
#   ./check.sh --full   # adds the integration-test suite (slow; matches `make all`)
#
# Do not add gate logic here. Add it to the Makefile or a script under
# scripts/ so the pre-push hook and CI pick it up automatically.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

MODE="${1:-check}"

case "$MODE" in
  --fix)
    echo "▶ auto-fixing formatting + imports"
    if command -v goimports >/dev/null 2>&1; then
      goimports -w -local github.com/CaliLuke/loom $(go list -f '{{.Dir}}' ./...)
    else
      echo "  goimports not on PATH; skipping (install: go install golang.org/x/tools/cmd/goimports@latest)"
    fi
    if command -v golangci-lint >/dev/null 2>&1; then
      golangci-lint run --fix || true
    fi
    exec make lint test
    ;;
  --full)
    exec make all
    ;;
  check|"")
    exec make lint test
    ;;
  *)
    echo "usage: $0 [--fix | --full]" >&2
    exit 2
    ;;
esac
