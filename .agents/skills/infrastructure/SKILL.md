---
name: infrastructure
description: Infrastructure reference for Podman, PostgreSQL, TypeDB, container operations, and datastore troubleshooting.
---

# Infrastructure & Datastores

Detailed reference for Podman, PostgreSQL, TypeDB, pgvector/pgai, and container setup. Consult this skill when working with infrastructure, debugging datastore issues, or setting up services.

## Always-On Services Assumption

Assume Postgres and TypeDB are already running during normal local development.

Health probe:

```bash
~/.typedb/typedb console --tls-disabled --address=localhost:1729 --username=admin --password=password --command="database list"
```

## Podman Compose (local services)

- Compose files: `docker-compose.yml` (Postgres + TypeDB) and `docker-compose.postgres.yml` (Postgres only). These are Podman-friendly (no Docker-only extensions).
- Start: `podman compose -f docker-compose.yml up -d` (or `... -f docker-compose.postgres.yml up -d` for DB-only). Stop: `podman compose -f docker-compose.yml down`.
- Volumes: named `auto-k-server_postgres_data` and `auto-k-server_typedb_data`. Postgres 18 mounts `/var/lib/postgresql`; TypeDB data lives under `/opt/typedb/server/data` in the container.
- Health/ports: Postgres `8002->5432`; TypeDB `1729` (gRPC) and `8000` (HTTP). TypeDB healthcheck is a simple `/dev/tcp` probe.

## PostgreSQL Connection Pooling

The Go server uses `pgx` connection pool directly.

**Production setup:** Connect through PgBouncer (port 6432).

### Python SQLAlchemy Pooling (Deprecated)

Controlled by `AUTO_K_PG_USE_NULLPOOL`:

| Mode           | Setting                                 | Pooling By | Use Case                               |
| -------------- | --------------------------------------- | ---------- | -------------------------------------- |
| **Production** | `AUTO_K_PG_USE_NULLPOOL=true` (default) | PgBouncer  | Production with PgBouncer on port 6432 |
| **Local Dev**  | `AUTO_K_PG_USE_NULLPOOL=false`          | SQLAlchemy | Development without PgBouncer          |
| **Tests**      | N/A (hardcoded `use_pool=True`)         | SQLAlchemy | Tests always use SQLAlchemy pooling    |

`PostgresSessionManager` is an async context manager - always use `async with` for automatic cleanup:

```python
async with PostgresSessionManager(dsn, use_pool=True) as manager:
    async with manager.session() as session:
        # do work
# connections automatically closed
```

## Pgvector & pgai (semantic similarity) — WIP

> **Status: Local dev kludge, not production-ready.** The embedding pipeline works for local testing but has significant gaps before production use.

Postgres has **pgvector** and **pgai** extensions enabled. Artifact versions get embedding vectors via the pgai vectorizer worker, stored in `artifact_versions_embedding_store`.

- **Embeddings table**: `artifact_versions_embedding_store` joined to `artifact_versions` on `id`.
- **API endpoints**: `GET /projects/{id}/graph/nodes/{display_id}/similar`, `POST /projects/{id}/graph/similarity/search`
- **Docs**: `docs/versioning.md` and `docs/semantic-similarity.md`

**pgai repo:** `/Users/luca/code/pgai` ([GitHub](https://github.com/CaliLuke/pg-rust-ai)) — our pgai extension and Rust vectorizer worker.

**Filing issues / improvements:** Drop markdown files in `/Users/luca/code/pgai/plans/` when you encounter bugs, integration friction, or have design proposals for the embedder pipeline.

### Architecture

- **SQL infrastructure**: Vendored from pgai repo into `src/vendor/pgai/setup.sql`. The Alembic migration calls `install_pgai_sql()` which executes this via raw DBAPI cursor (SQLAlchemy's `text()` can't handle PostgreSQL `format()` `%I/%L/%s` specifiers).
- **Rust worker**: Built from `/Users/luca/code/pgai` via `Dockerfile.worker`. Spawned by `main.py` lifespan using `VECTORIZER_WORKER_BINARY` config. CLI: `--db-url`, `--poll-interval N` (seconds), `--once`, `--vectorizer-ids`. Log level via `RUST_LOG` env var.
- **Worker tracking**: `setup.sql` includes `ai.vectorizer_worker_process` and `ai.vectorizer_worker_progress` tables. Design doc: `/Users/luca/code/pgai/plans/WORKER_TRACKING_DESIGN.md`.

**Re-syncing vendored SQL:** When `setup.sql` changes in the pgai repo:

```bash
cp /Users/luca/code/pgai/extension/sql/setup.sql src/vendor/pgai/setup.sql
```

**Current local setup:**

- Model: `embeddinggemma:300m` (768 dims, ~2s/doc on Apple Silicon)
- Worker: Rust binary from pgai repo

**Known issues / TODO:**

- No production embedding model chosen — embeddinggemma:300m is for local speed, not quality
- Vectorizer config is baked into Alembic migration — no clean way to swap models without dropping/recreating
- Stale `idle in transaction` from killed workers block the queue — no automatic recovery
- Duplicate detection implemented via `graph_check` (`duplicate-artifacts` rule) using hybrid RRF (vector + full-text)
- No HNSW index on embedding column — linear scan only, fine for small datasets
- Similarity endpoints not wired into any UI or MCP tool workflow

### Vectorizer worker logs

The pgai vectorizer worker logs are captured to `logs/logs.db` in the `vectorizer_logs` table:

```bash
# Recent activity
sqlite3 logs/logs.db "SELECT timestamp, level, message FROM vectorizer_logs ORDER BY id DESC LIMIT 20"

# Errors only
sqlite3 logs/logs.db "SELECT timestamp, message FROM vectorizer_logs WHERE level = 'error' ORDER BY id DESC LIMIT 10"

# Embedding progress (items processed per run)
sqlite3 logs/logs.db "SELECT timestamp, message FROM vectorizer_logs WHERE message LIKE '%finished%' OR message LIKE '%Items pulled%' ORDER BY id DESC LIMIT 20"
```

### Vectorizer worker troubleshooting

**Worker hangs / no items processed:**

1. Check for stale transactions: `SELECT pid, state, query_start FROM pg_stat_activity WHERE state = 'idle in transaction' AND query LIKE '%vectorizer%';`
2. Kill them: `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle in transaction' AND query LIKE '%vectorizer%';`
3. Restart the worker

**Running the Rust worker manually:**

```bash
# Build the worker (if not already built)
cd /Users/luca/code/pgai && cargo build --release

# Run once against local database
OLLAMA_HOST="http://localhost:11434" RUST_LOG=info \
  /Users/luca/code/pgai/target/release/pgai-worker \
  --db-url "postgresql://admin:password@localhost:8002/auto_k" \
  --vectorizer-ids 29 --once
```

## TypeDB Inspection (for debugging)

Use the inspection script to query TypeDB data directly:

```bash
# List all databases
uv run python scripts/inspect_typedb.py list

# Count entities by type in a database
uv run python scripts/inspect_typedb.py count project_<uuid_hex>

# List all artifacts with display IDs and titles
uv run python scripts/inspect_typedb.py artifacts project_<uuid_hex>

# Run arbitrary TypeQL query
uv run python scripts/inspect_typedb.py query project_<uuid_hex> 'match $x isa persona; select $x;'
```

Database naming: Production databases use `project_<uuid_hex>` (no dashes). Tests should set `AUTO_K_TYPEDB_DB_PREFIX=tmp_test_` to use `tmp_test_<uuid_hex>` instead.

**DANGER**: The test fixtures `ensure_project_db` and `typedb_project_store` auto-drop databases after use. **NEVER** use them with real project IDs.

For safe inspection of existing databases, use `inspect_project_db` instead:

```python
from tests.utils.typedb_temp_db import inspect_project_db
with inspect_project_db(manager, project_id) as (db_name, store):
    # Read-only operations - does NOT create or drop anything
    pass
```

## TypeDB Troubleshooting

- If the probe (`~/.typedb/typedb console --tls-disabled --address=localhost:1729 --username=admin --password=password --command="database list"`) fails or logs show checkpoint corruption (`[STO10] Failed to recover from checkpoint...`), the data volume is likely corrupted.
- Recovery (destructive):
  1. `podman stop auto-k-server-typedb-1 && podman rm auto-k-server-typedb-1`
  2. `podman volume rm auto-k-server_typedb_data` (wipes TypeDB data)
  3. `podman compose -f docker-compose.yml up -d typedb`
  4. Re-probe with the console command and re-seed any required databases.
- Avoid corruption by allowing clean shutdowns. Back up/export before upgrades.

### TypeDB stack overflow crashes (FIXED)

TypeDB 3.x versions (including 3.7.0-rc0 and 3.7.2) could crash with `tokio-runtime-worker` stack overflow during query **compilation** (not execution).

**Root cause (now fixed):** TypeBridge's `__in` lookup previously generated **deeply nested binary OR** expressions like `((((a or b) or c) or d) or e)` instead of flat disjunctions `{a} or {b} or {c}`. With 75+ values, the ~75 levels of nesting caused the recursive query planner to overflow its stack.

**Fix applied:** This was fixed in type-bridge 1.2.3+ (see [issue #76](https://github.com/ds1sqe/type-bridge/issues/76)). No workarounds are needed.

### TypeDB log capture (for crash debugging)

TypeDB container logs are captured to SQLite for crash analysis:

```bash
# Start log capture in background
uv run python scripts/capture_typedb_logs.py &

# Query captured logs
sqlite3 logs/typedb.db "SELECT timestamp, message FROM logs WHERE level = 'ERROR' ORDER BY id DESC LIMIT 50"

# Find stack overflow crashes
sqlite3 logs/typedb.db "SELECT raw_line FROM logs WHERE message LIKE '%overflow%' OR message LIKE '%Backtrace%'"

# Log stats by level
sqlite3 logs/typedb.db "SELECT level, COUNT(*) FROM logs GROUP BY level"
```

To get full Rust backtraces on crash, ensure `RUST_BACKTRACE=1` is set in `docker-compose.yml` under the typedb service environment.

## Argo CD Failure Triage Loop

Use this exact loop before changing code for a CI/CD failure:

1. Confirm live failures in Argo:
   - `kubectl -n argo-ci get wf`
   - Identify the latest failed workflow for the affected service.
2. Find the failing node/step:
   - `kubectl -n argo-ci get wf <name> -o json | jq ...`
   - Classify step (`preflight`, `clone`, `lint`, `test`, `build`, deploy, `onExit`).
3. Pull pod logs for that node and capture exact error lines.
4. Classify failure type:
   - Pipeline/setup issue: version pin drift, missing secret/config, build flag mismatch, registry/auth/network infra.
   - Code issue: deterministic compile/test/runtime defect in repo source.
5. If pipeline/setup issue:
   - File Linear issue(s) with workflow ID, step name, exact error, remediation, acceptance criteria.
6. If code issue:
   - Fix code and rerun repo quality gates.
7. Verify closure:
   - Confirm the next Argo run moves from `Failed` to `Succeeded`.

Do not label a failure as pipeline-related without evidence from the failing Argo step logs.

## Container Setup

- `Dockerfile.local`: multi-stage Rust FFI → Go → Python+Go runtime
- `Dockerfile.rustffi` + `scripts/build-rustffi.sh`: pre-built Rust FFI image, rebuild only when go-typeql changes
- `scripts/build-container.sh`: copies go-typeql source, builds Go+Python using pre-built Rust FFI
- `entrypoint.local.sh`: Go on :8000 (primary entry point), Python on :8001 (Alembic migrations only)
- Port: host 8000 → container 8000
- Podman doesn't support `additional_contexts` — use rsync into `.build-go-typeql/` temp dir instead

### Python Dev Container (Deprecated)

For development, the server can run in a Podman container. Files: `Dockerfile.dev` + `docker-compose.dev.yml` + `.dockerignore`

```bash
# Build and start
podman compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build

# Restart after code changes
podman restart auto-k-server-dev

# View logs
podman logs -f auto-k-server-dev

# Stop
podman compose -f docker-compose.yml -f docker-compose.dev.yml down
```

**How it works:**

- `Dockerfile.dev` installs deps + BAML in a cached layer, copies `src/` only for the build
- At runtime, `src/` is bind-mounted over the installed copy, so code edits are live after a restart
- `logs/`, `data/` are also bind-mounted for persistence
- TypeDB and Ollama are accessed via `host.containers.internal` (running on host, not in compose)
- Postgres is accessed via compose networking (`postgres:5432`)

**When to rebuild vs restart:**

- Code changes in `src/` → `podman restart auto-k-server-dev`
- Dependency changes (`pyproject.toml`, `uv.lock`) → rebuild with `--build`
- BAML schema changes → rebuild with `--build`
