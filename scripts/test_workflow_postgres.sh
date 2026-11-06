#!/usr/bin/env bash
# Explicit, isolated PostgreSQL acceptance for the durable workflow roles.
set -euo pipefail

WORKFLOW_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Latest stable upstream PostgreSQL and official multi-platform manifest,
# verified 2026-11-05. PostgreSQL 19 was still a release candidate.
WORKFLOW_IMAGE='postgres:18.6@sha256:bd5dfa494a5a259704365813dbfa5fa6b3e69be7348d43b547f540c730478583'
WORKFLOW_CONTAINER=''

cleanup() {
  if [[ -n "$WORKFLOW_CONTAINER" ]]; then
    docker rm -f -v "$WORKFLOW_CONTAINER" >/dev/null
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

command -v docker >/dev/null 2>&1 || {
  echo 'test-workflow-postgres: Docker is required' >&2
  exit 1
}
WORKFLOW_CONTAINER="$(docker run -d --rm -e POSTGRES_PASSWORD=workflow-acceptance -e POSTGRES_DB=workflow_acceptance -p 127.0.0.1::5432 "$WORKFLOW_IMAGE")"
ready=''
for ((attempt=0; attempt<100; attempt++)); do
  if docker exec "$WORKFLOW_CONTAINER" pg_isready -h 127.0.0.1 -U postgres -d workflow_acceptance >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 0.2
done
if [[ -z "$ready" ]]; then
  echo 'test-workflow-postgres: database readiness timed out' >&2
  exit 1
fi
binding="$(docker port "$WORKFLOW_CONTAINER" 5432/tcp)"
port="${binding##*:}"
docker exec "$WORKFLOW_CONTAINER" psql -U postgres -d workflow_acceptance -Atc 'select version()'
cd "$WORKFLOW_ROOT"
LOOM_WORKFLOW_POSTGRES_DSN="postgres://postgres:workflow-acceptance@127.0.0.1:$port/workflow_acceptance?sslmode=disable" \
LOOM_WORKFLOW_REQUIRE_POSTGRES=1 \
go test -race ./execution/selvedge -run '^TestWorkflowPostgres(Admission|TwoProcess)$' -count=1 -timeout 2m
