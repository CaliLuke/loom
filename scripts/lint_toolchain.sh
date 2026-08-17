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

require_line 'GOLANGCI_LINT_VERSION?=v2.12.2'
require_line 'PROTOC_GEN_GO_VERSION?=v1.36.12'
require_line 'PROTOC_GEN_GO_GRPC_VERSION?=v1.6.2'
require_line 'PROTOC_VERSION=35.1'
require_line 'STATICCHECK_VERSION?=v0.8.0-rc.1'

if grep -Eq 'google\.golang\.org/(protobuf/cmd/protoc-gen-go|grpc/cmd/protoc-gen-go-grpc)@latest' "$makefile"; then
  echo "$makefile: protobuf generators must use pinned versions, not @latest" >&2
  exit 1
fi

if grep -Eq 'honnef\.co/go/tools/cmd/staticcheck@latest' "$makefile"; then
  echo "$makefile: staticcheck must use a pinned version, not @latest" >&2
  exit 1
fi
