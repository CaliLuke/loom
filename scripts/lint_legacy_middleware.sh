#!/usr/bin/env bash
set -euo pipefail

forbidden_paths=(
  middleware
  http/middleware/capture.go
  http/middleware/log.go
  http/middleware/requestid.go
  http/middleware/trace.go
  http/middleware/xray
  grpc/middleware/log.go
  grpc/middleware/requestid.go
  grpc/middleware/trace.go
  grpc/middleware/xray
)

status=0
for path in "${forbidden_paths[@]}"; do
  if [[ -e "$path" ]]; then
    echo "$path: legacy middleware surface must be removed" >&2
    status=1
  fi
done

if rg -n 'AsLoomMiddlewareLogger|GRPCClientLogOption' clue/log --glob '*.go'; then
  echo "clue/log: legacy middleware adapters or aliases must be removed" >&2
  status=1
fi

exit "$status"
