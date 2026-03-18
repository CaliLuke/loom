---
name: benchmarking
description: Run and extend the Auto-K performance harness. Use when benchmarking TypeDB graph hot paths, recording long-term latency history in perf/history.sqlite, checking for regressions, or adding new benchmark scenarios and suites.
---

# Benchmarking

Use this skill whenever the task is about performance baselines, regressions, benchmark history, or adding new tracked benchmark scenarios.

## Quick Run

Prefer the built-in CLI instead of ad hoc scripts:

```bash
LOG_LEVEL=warn \
MACOSX_DEPLOYMENT_TARGET=26.2 \
CGO_LDFLAGS="$(./scripts/build-rustffi.sh --print-host-cgo-ldflags)" \
CGO_ENABLED=1 \
GOFLAGS="-tags=cgo,typedb,typedb_prebuilt" \
go run ./cmd/server perf --db-path perf/history.sqlite --note "what changed"
```

Default behavior:

* runs the tracked integration benchmark suite
* appends the run to `perf/history.sqlite`
* compares each scenario to the previous completed run
* fails on regressions above `--threshold-pct` unless `--fail-on-regression=false`

## Current Tracked Surface

The tracked suite lives in `internal/perf/typedb_suite.go` and now covers a broader TypeDB performance surface:

* filtered graph reads with and without relations
* full graph reads with and without relations
* work-slice graph reads
* search reads with and without relations
* raw and formatted node-details reads
* Mermaid-formatted search output
* aggregate queries such as `CountArtifactsByType`
* coverage checks via `RunChecks`
* small write-path benchmarks for `CreateNodes` and `ConnectNodes`

These scenarios exist to catch regressions like AUTOK-238 before they ship.

## History Queries

Inspect recent runs:

```bash
sqlite3 perf/history.sqlite "SELECT id, started_at, git_branch, git_commit, status, notes FROM perf_runs ORDER BY id DESC LIMIT 10"
```

Inspect per-scenario medians and regressions:

```bash
sqlite3 perf/history.sqlite "SELECT run_id, suite, benchmark_name, median_ns/1000000.0 AS median_ms, p95_ns/1000000.0 AS p95_ms, delta_pct, regression FROM perf_measurements ORDER BY run_id DESC, suite, benchmark_name"
```

## Extending The Harness

When you add a new benchmark:

1. Prefer adding it to `internal/perf` as a named scenario, not as a one-off shell loop.
2. Reuse deterministic fixtures and keep setup outside the timed section when practical.
3. Assert the scenario result shape so a broken query fails loudly instead of recording junk latency.
4. Record enough samples to compare medians meaningfully; keep warmups explicit.
5. Update `perf/history.sqlite` by running the command once after the new scenario lands.

Design rules:

* keep suites small and purpose-built
* benchmark the real integration seam when possible
* store raw samples plus summary stats
* compare against the previous completed run, not hand-written thresholds alone

## Validation

Before finishing:

* run the relevant unit tests for `internal/perf`
* run `./scripts/run-golangci.sh`
* run the exact pre-push Go test command
* run `go-server perf` at least once when the benchmark surface changes so the SQLite history stays current
