#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE_FILE="${ROOT_DIR}/jsonrpc/integration_tests/.goa_source_mode"
LOCAL_GOA_DIR="${GOA_DIR:-${ROOT_DIR}}"

usage() {
  cat <<EOF
Usage: $(basename "$0") <local|remote|status>

Modes:
  local   Point JSON-RPC temp-module generation at a local Goa checkout (${LOCAL_GOA_DIR} by default)
  remote  Use the default pinned remote checkout flow
  status  Print the configured source mode

Environment:
  GOA_DIR   Override the local Goa checkout path used by local mode
EOF
}

set_local() {
  if [[ ! -f "${LOCAL_GOA_DIR}/go.mod" ]]; then
    echo "local Goa checkout not found at ${LOCAL_GOA_DIR}" >&2
    exit 1
  fi
  printf 'local %s\n' "${LOCAL_GOA_DIR}" > "${MODE_FILE}"
  echo "goa source mode: local (${LOCAL_GOA_DIR})"
}

set_remote() {
  printf 'remote\n' > "${MODE_FILE}"
  echo "goa source mode: remote"
}

show_status() {
  if [[ ! -f "${MODE_FILE}" ]]; then
    echo "goa source mode: remote (default)"
    exit 0
  fi
  case "$(cut -d' ' -f1 < "${MODE_FILE}")" in
    local)
      echo "goa source mode: $(cat "${MODE_FILE}")"
      ;;
    remote)
      echo "goa source mode: remote"
      ;;
    *)
      echo "goa source mode: invalid ($(cat "${MODE_FILE}"))"
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
