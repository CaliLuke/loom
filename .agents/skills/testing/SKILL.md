---
name: testing
description: Reference for Go unit tests and Python integration-test patterns, fixtures, and quality gates used in Auto-K.
---

# Testing Reference

Detailed testing patterns for Go and Python integration tests. Consult this skill when writing tests, debugging test failures, or understanding fixture design.

## Go Quality Gates (Current Repo)

- Never bypass safeguards (`NOTEST=1`, `NOFIX=1`, `--no-verify`) unless the user explicitly asks.
- Run `golangci-lint run ./...` before pushing.
- Run `go test ./...` before pushing.
- Treat pre-commit or pre-push failures as required fixes, not optional warnings.
- For deployment verification, reproduce via `curl` against staging and then verify SigNoz signals in the same window.

## Go Tests (Primary)

```bash
cd go-server && go test ./...

# With verbose output
cd go-server && go test -v ./...

# Specific package
cd go-server && go test -v ./internal/graph/...
```

Go unit tests cover graph operations, auth, events, slugify, and transport layer.

## Python Integration Tests (Migration Validation)

Python integration tests validate Go behavior via HTTP. These tests are the contract — if they fail, the Go code is wrong.

### Test Organization

**Never create throwaway test scripts in `scripts/`** to verify functionality. Instead:

1. **Unit tests** (`src/tests/unit/`) - For testing isolated logic without external dependencies
2. **Integration tests** (`src/tests/integration/`) - For testing with real TypeDB/Postgres

When testing new TypeDB functions or MCP tools:

- Add tests to existing test classes that already have database fixtures set up
- Reuse `e2e_mcp_context` fixture for MCP tool tests (creates one DB per test class)
- Check `src/tests/integration/mcp/test_mcp_tools.py` for patterns

### Continent: TypeDB Test Fixture Design

TypeDB schema application is expensive (~800ms). Integration tests share a single **session-scoped database** to avoid creating one per test:

- **`continent_db`** — Shared database where data accumulates. **All graph/MCP integration tests use this.**

**The core principle: continent is the default, no exceptions** (aside from infrastructure tests listed below). If a test "needs" an empty database, it actually needs a fresh `project_id` — which `continent_store` already provides via project-scoped isolation.

**Design rules for integration tests:**

1. **Tests MUST be idempotent** — never depend on execution order or global counts. Query by known IDs/display_ids, not `len(results) == N`.
2. **Isolate via `project_id`, don't recreate DBs** — each test gets a unique `project_id`. All queries go through TypeBridge or project-scoped schema functions, so each test sees only its own data.
3. **Use `>= N` not `== N`** — if asserting counts on a shared DB, other tests may have added data. Better: scope counts to the test's project_id.
4. **Only function-scoped `test_client` owns `reset_models()`** — it calls `DROP SCHEMA CASCADE` + recreate on every test. Class-scoped (`test_client_class`) and session-scoped (`test_client_session`) fixtures use `build_client_no_reset` and rely on tables already existing. This prevents higher scopes from nuking data out from under each other. All scopes create fresh Postgres entities (account, user, project) per test with random IDs (`secrets.randbelow()`). Never use hardcoded IDs.
5. **TypeDB is session-scoped** — `continent_db` persists across all tests. Use `patch("...project_database_name", return_value=db_name)` to route any project_id to the shared DB.
6. **No raw TypeQL in tests** — all graph queries must go through TypeBridge or project-scoped schema functions. Raw TypeQL bypasses project isolation and couples tests to query syntax. If a test can't be project-scoped, that's a smell — either the test is wrong or the schema function needs a project_id parameter.
7. **Only infrastructure tests get their own DB** — migrations tests (test migration machinery), kill_switch tests (intentionally crash driver), regex tests (custom schema). These test DB infrastructure, not graph behavior.
8. **Never use ClassVar flags to gate seeding** — `_graph_seeded: ClassVar[bool]` is a buggy reimplementation of pytest fixture scoping that causes cross-datastore desync (Postgres nuked mid-class while TypeDB retains stale data). Use fixture scoping instead: a class-scoped fixture with `test_client_class` seeds once and keeps Postgres + TypeDB in sync for the class lifetime.

**Migration checklist (converting own-DB tests to continent):**

1. Replace `graph_store` / `typedb_project_store` / `typedb_db` with `continent_store`
2. Verify all queries are project-scoped (through TypeBridge or schema functions)
3. Replace raw TypeQL with TypeBridge calls where possible
4. Update count assertions: use `>= N` or scope to project_id
5. Remove any "empty state" assertions that assumed a clean DB — a fresh project_id is the clean state

### Console Log Suppression in Tests

Console logs are **suppressed by default** during pytest runs for clean, compact output. Logs still go to SQLite and file sinks.

```bash
# Default: quiet output
uv run pytest ...

# Enable console logs for debugging
LOG_CONSOLE_ENABLED=true uv run pytest ...
```

This is detected via the `PYTEST_CURRENT_TEST` environment variable that pytest sets automatically.

## Git Hooks (automatic enforcement)

The repo uses `.githooks/` for quality gates. Enable with: `git config core.hooksPath .githooks`

| Hook           | What runs                                 | Time    | Behavior                                         |
| -------------- | ----------------------------------------- | ------- | ------------------------------------------------ |
| **pre-commit** | Ruff, Vulture, Mypy, Ty                   | ~4s     | Generates `FIXME.md` on failure, commit proceeds |
| **pre-push**   | Full `uv run lint` (Pylint, Radon, tests) | ~1-2min | Blocked if `FIXME.md` exists                     |

**Flow:** Commit, inspect `FIXME.md`, fix issues, then push. Do not bypass checks unless explicitly instructed by the user.

### Manual commands

- Lints only: `uv run lint --notest`
- Full suite: `uv run lint` (lints + tests, generates `FIXME.md` on failure)
- Tests only: `uv run pytest`

### Go Server Test Fixture

The `go_server` pytest fixture (session-scoped, autouse) auto-starts the binary on port 8091. `GO_SERVER_URL` env var controls the Python client target. The Go server accepts a `db_name` query param so Python can pass the correct TypeDB database name (important for test prefix compatibility).

## Parallel Agent Run Protocol

Use this when multiple workers edit the same branch/worktree:

1. Define non-overlapping ownership lanes (file list/prefix per worker).
2. During worker edit phase, do not run full-repo checks (`golangci-lint run ./...`, `go test ./...`, `git commit`).
3. Workers only do focused edits, file-local formatting, and report changed files.
4. Integrate sequentially in one coordinator pass:
   - `goimports -w` on integrated touched files
   - `golangci-lint run ./...`
   - `go test ./...`
   - commit integration result
5. If a lane is blocked by another lane’s WIP, pause and land dependency lane first.
6. If workers are interrupted, treat outputs as WIP and reconcile explicitly before centralized validation.
