#!/usr/bin/env bash
# Runs the pulse test suites against real Redis servers (issue #383).
#
# With LOOM_PULSE_REDIS_ADDR set, the suites run once against that server.
# CI uses this mode with a service container. Otherwise the script starts one
# Docker container per version in LOOM_PULSE_REDIS_VERSIONS, runs the suites
# against it and removes it.
#
# The tests flush Redis databases 1 to 3, so point LOOM_PULSE_REDIS_ADDR at a
# disposable server only. The tests refuse a non-loopback address unless
# LOOM_PULSE_REDIS_ALLOW_REMOTE=1. Packages run one at a time (-p 1) because
# pub/sub channels are shared by all databases of a server.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSIONS="${LOOM_PULSE_REDIS_VERSIONS:-6.2 7.4}"
PACKAGES=(./pulse/...)

CONTAINER=""
GO_PID=""

cleanup() {
  if [[ -n "$CONTAINER" ]]; then
    docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  fi
}

# terminate stops the running go test, its test binaries and the container.
terminate() {
  if [[ -n "$GO_PID" ]]; then
    pkill -TERM -P "$GO_PID" 2>/dev/null || true
    kill -TERM "$GO_PID" 2>/dev/null || true
  fi
  cleanup
  exit 143
}

trap cleanup EXIT
trap terminate INT TERM

# run_suites runs go test in the background and waits, so a signal to this
# script reaches the trap and stops go test too.
run_suites() {
  local addr="$1"
  echo "▶ pulse suites against Redis at $addr"
  (cd "$ROOT" && LOOM_PULSE_REDIS_ADDR="$addr" \
    exec go test -race -shuffle=on -count=1 -p 1 -timeout 20m "${PACKAGES[@]}") &
  GO_PID=$!
  local status=0
  wait "$GO_PID" || status=$?
  GO_PID=""
  return "$status"
}

if [[ -n "${LOOM_PULSE_REDIS_ADDR:-}" ]]; then
  run_suites "$LOOM_PULSE_REDIS_ADDR"
  exit 0
fi

command -v docker >/dev/null 2>&1 || {
  echo "test-pulse-redis: docker is required unless LOOM_PULSE_REDIS_ADDR is set" >&2
  exit 1
}

failed=()
for version in $VERSIONS; do
  CONTAINER="$(docker run -d --rm -p 127.0.0.1::6379 "redis:$version")"
  ready=""
  for _ in $(seq 1 50); do
    if docker exec "$CONTAINER" redis-cli ping 2>/dev/null | grep -q PONG; then
      ready=1
      break
    fi
    sleep 0.2
  done
  if [[ -z "$ready" ]]; then
    echo "test-pulse-redis: redis:$version not ready" >&2
    exit 1
  fi
  port="$(docker port "$CONTAINER" 6379/tcp | head -n 1 | sed 's/.*://')"
  echo "▶ redis:$version ($(docker exec "$CONTAINER" redis-server --version | sed -E 's/.* v=([^ ]+).*/\1/'))"
  if ! run_suites "127.0.0.1:$port"; then
    failed+=("$version")
  fi
  docker rm -f "$CONTAINER" >/dev/null
  CONTAINER=""
done

if ((${#failed[@]} > 0)); then
  echo "test-pulse-redis: failed against Redis ${failed[*]}" >&2
  exit 1
fi
echo "test-pulse-redis: passed against Redis $VERSIONS"
