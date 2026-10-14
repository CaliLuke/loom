#!/usr/bin/env bash
# Keep complete package test sets together so shuffled-order coverage survives CI partitioning.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
lane="${1:-all}"
timeout="${2:-40m}"
case "$lane" in
  all) exec go test -race -shuffle=on -count=1 -timeout "$timeout" ./... ;;
  http|grpc|generator|remaining) ;;
  *) echo "unknown race lane: $lane" >&2; exit 1 ;;
esac

inventory="$(go list -race ./...)"
http_package=github.com/CaliLuke/loom/http/codegen
grpc_package=github.com/CaliLuke/loom/grpc/codegen
generator_package=github.com/CaliLuke/loom/codegen/generator
for owner in "$http_package" "$grpc_package" "$generator_package"; do
  if ! grep -Fxq "$owner" <<<"$inventory"; then
    echo "missing dedicated race package: $owner" >&2
    exit 1
  fi
done

packages=()
while IFS= read -r package; do
  case "$package" in
    "$http_package") package_lane=http ;;
    "$grpc_package") package_lane=grpc ;;
    "$generator_package") package_lane=generator ;;
    *) package_lane=remaining ;;
  esac
  if [[ "$package_lane" == "$lane" && -n "$package" ]]; then
    packages+=("$package")
  fi
done <<<"$inventory"
if [[ "${#packages[@]}" == 0 ]]; then
  echo "empty race lane: $lane" >&2
  exit 1
fi
exec go test -race -shuffle=on -count=1 -timeout "$timeout" "${packages[@]}"
