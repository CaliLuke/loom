#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE_FILE="${ROOT_DIR}/jsonrpc/integration_tests/.loom_source_mode"
LOCAL_LOOM_DIR="${LOOM_DIR:-${ROOT_DIR}}"

usage() {
  cat <<EOF
Usage: $(basename "$0") <local|remote|status>

Modes:
  local   Point JSON-RPC temp-module generation at a local Loom checkout (${LOCAL_LOOM_DIR} by default)
  remote  Use the default pinned remote checkout flow
  status  Print the configured source mode

Environment:
  LOOM_DIR  Override the local Loom checkout path used by local mode
EOF
}

set_local() {
  if [[ ! -f "${LOCAL_LOOM_DIR}/go.mod" ]]; then
    echo "local Loom checkout not found at ${LOCAL_LOOM_DIR}" >&2
    exit 1
  fi
  printf 'local %s\n' "${LOCAL_LOOM_DIR}" > "${MODE_FILE}"
  echo "loom source mode: local (${LOCAL_LOOM_DIR})"
}

set_remote() {
  printf 'remote\n' > "${MODE_FILE}"
  echo "loom source mode: remote"
}

show_status() {
  if [[ ! -f "${MODE_FILE}" ]]; then
    echo "loom source mode: remote (default)"
    exit 0
  fi
  case "$(cut -d' ' -f1 < "${MODE_FILE}")" in
    local)
      echo "loom source mode: $(cat "${MODE_FILE}")"
      ;;
    remote)
      echo "loom source mode: remote"
      ;;
    *)
      echo "loom source mode: invalid ($(cat "${MODE_FILE}"))"
      exit 1
      ;;
  esac
}

main() {
  if [[ $# -ne 1 ]]; then
    usage >&2
    exit 1
  fi

  case "$1" in
    local)
      set_local
      ;;
    remote)
      set_remote
      ;;
    status)
      show_status
      ;;
    *)
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
