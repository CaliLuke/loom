#!/usr/bin/env bash
set -euo pipefail

makefile="${1:-Makefile}"

require_line() {
  local expected="$1"
  if ! grep -Fqx "$expected" "$makefile"; then
    echo "$makefile: missing pinned toolchain declaration: $expected" >&2
    return 1
  fi
}

require_line 'PROTOC_GEN_GO_VERSION?=v1.36.11'
require_line 'PROTOC_GEN_GO_GRPC_VERSION?=v1.6.2'

if grep -Eq 'google\.golang\.org/(protobuf/cmd/protoc-gen-go|grpc/cmd/protoc-gen-go-grpc)@latest' "$makefile"; then
  echo "$makefile: protobuf generators must use pinned versions, not @latest" >&2
  exit 1
fi
