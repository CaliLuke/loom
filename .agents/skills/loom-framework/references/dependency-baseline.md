# Dependency baseline

Review date: 2026-11-14. The inventory covers the root Go requirements, seven
active nested integration/fixture modules, Makefile tools, embedded OpenAPI
consumer/linter pins, Node/AJV, CI action majors, and Redis/PostgreSQL images.
Historical reproduction modules and proof toolchains remain reproducibility
inputs rather than active application dependency baselines.

## Adopted updates

| Dependency | Previous | Selected | Relevant upstream changes |
|---|---|---|---|
| Go | 1.27.3 | 1.27.4 | Compiler/runtime corrections and security fixes, including net/http. |
| libopenapi | v0.42.4 | v0.42.5 | Header examples no longer inherit schema examples; reference identity, bundling and renderer corrections. |
| pgx | v5.12.0 | v5.13.0 | Reject malformed row counts without panicking; do not classify response-side connection failure as safe to retry. |
| modernc/libc | v1.80.2 | v1.81.1 | Implements Darwin fsctl used by SQLite network-volume locking; TLS checkpoint, allocator and BSD corrections. |
| Node | 26.13.0 | 26.13.1 | HTTP/2, TLS and other security corrections in the contract-test runtime. |
| TanStack Query | 5.106.1 | 5.107.1 | Updates the generated TypeScript consumer's query-core dependency. |
| PostgreSQL | 18.6 | 18.7 | Stable server security and correctness update; PostgreSQL 19 remains a release candidate. |

Sources: [Go](https://go.dev/doc/devel/release#go1.27.4),
[libopenapi](https://github.com/pb33f/libopenapi/releases/tag/v0.42.5),
[Vacuum](https://github.com/daveshanley/vacuum/releases/tag/v0.41.0),
[pgx changelog](https://github.com/jackc/pgx/blob/v5.13.0/CHANGELOG.md),
[libc changes](https://gitlab.com/cznic/libc/-/compare/v1.80.2...v1.81.1),
[Node](https://nodejs.org/en/blog/release/v26.13.1),
[TanStack Query](https://github.com/TanStack/query/releases/tag/%40tanstack%2Freact-query%405.107.1),
[PostgreSQL](https://www.postgresql.org/docs/release/18.7/).

The Go directive is aligned in active fixture modules, the caller-workspace
fixture and current user recipes; CI reads the root go.mod. PostgreSQL uses the
official multi-platform manifest
`sha256:2ea8630df176ad090ec92136dee850397c5af8781e4bcf2e4162593691c61bbc`.
The pgx tag is verified through the Go proxy and upstream tag/changelog because
GitHub's latest Release object still points to v5.11.0.

libc must be reviewed as part of the generated SQLite runtime, not assumed safe
from its version number. The upstream longjmp ABI warning also exists in the
previous v1.80.2 source; it is not a newly introduced v1.81.1 incompatibility.
The selected pair remains SQLite v1.62.0 with libc v1.81.1. Consumer checks cover
compilation and the durable workflow SQLite lifecycle, including process restart;
network-volume fsctl behavior requires an actual AFP/SMB mount and is not claimed
as locally exercised.

## Removed and retained exceptions

- Removed the header source-presence workaround for
  [libopenapi #695](https://github.com/pb33f/libopenapi/issues/695).
  Header examples now use the same high-level example owner as other wire
  examples. Existing absent/null, merged, named, reusable and rejected controls
  remain in `internal/openapiimport/producer_metadata_test.go`.
- Retained the Vacuum correctness snapshot after testing v0.41.0. The stable
  release includes the external-security fixes for
  [#1053](https://github.com/daveshanley/vacuum/issues/1053) and
  [#1056](https://github.com/daveshanley/vacuum/issues/1056), but silently accepts
  conflicting nested encodings. Both CLI/motor controls fail, and an unmodified
  released CLI accepts a standalone invalid document. This is reported in
  [#1066](https://github.com/daveshanley/vacuum/issues/1066). Return to stable when
  all existing positive and negative controls pass without weakening rules; see
  [OpenAPI validation](openapi-validation.md).
- Retained the go-redis fork. Latest upstream stable v9.23.0 predates merged
  [PR #4095](https://github.com/redis/go-redis/pull/4095); the existing exact
  fork pin already contains the hook-copy correction. Its provenance and return
  condition remain in `pulse/streaming/tla/read_shutdown/dependency_fetchability.md`.
- Retained the HTTP trace cancellation workaround:
  [Go #76669](https://github.com/golang/go/issues/76669) remains open and
  `TestHTTPTraceComposeDiscardsNewHookError` still reproduces the composition
  behavior on Go 1.27.4.
- Retained the miniredis struct shim. Latest v2.40.0 still omits the Lua library;
  [#180](https://github.com/alicebob/miniredis/issues/180) is closed but has a
  current reproduction. Issue closure alone is not evidence of released support.

The other root requirement pins, active fixture library pins, Selvedge v0.8.1,
SQLite v1.62.0, Redis 8.10.3, golangci-lint v2.15.0, protoc 36.2,
protoc-gen-go v1.36.12, protoc-gen-go-grpc v1.6.2, AJV 8.20.0,
Hey API 0.99.0/client-fetch 0.13.1, TypeScript 7.0.2 (generator API 6.0.2),
Zod 4.6.7, React 19.3.1 and oapi-codegen v2.8.2 are current stable pins.
Existing GitHub Action major tags already track the current release lines.

## Validation scope

The acceptance rule is unchanged import/example semantics and generated consumer
compilation, Vacuum validation without disabling rules, and working
SQLite/PostgreSQL durable workflow execution with the selected dependency graph.

- `TestImportProducerHeaderExamples`, its rejected-input controls and the SSE
  compilation case cover the removed workaround. Reusing the schema-only test
  with v0.42.4 and the simplified importer reproduces the unwanted second example;
  v0.42.5 passes. No assertion is weakened.
- Six representative rendered designs (distinct scopes, absent header example,
  merged header, named/reusable examples and SSE metadata) are byte-identical
  between parent `adf832bdb` and the candidate and across independent candidate
  processes. Both runs use Go 1.27.4. This is a bounded comparison, not an audit
  of the full exported-design corpus.
- Focused race tests cover `TestImportProducer.*` and the exhaustive grammar
  ledger in `internal/openapiimport`. The full `execution/selvedge` race suite
  passes on macOS/arm64 and Linux/arm64 using Go 1.27.4. On Linux, the ordinary
  example binaries are built first so cold compilation does not consume their
  one-minute test budget.
- `make test-workflow-postgres` exercises PostgreSQL 18.7 and pgx v5.13.0,
  including two-process interruption and recovery.
- `make openapi-contract`, run with Node 26.13.1, checks rendered JSON/YAML,
  invalid controls and generated Go/TypeScript consumers. The separately
  regenerated identity specimen also runs through
  `TestExternalSecurityDocumentsPassVacuum` with its supplied security document.
- `make generated-code-quality` regenerates the five maintained transport
  fixtures in temporary copies, then tests, vets and lints the generated code.
  Quality fixtures require generation before tidy; do not prune their manifests
  merely because ignored generated packages are absent from the source checkout.
- `make lint`, module verification and `git diff --check` cover source hygiene.
- `TestGeneratorUsesCallerWorkspace` covers `gen`, `example` and `vet` with
  both workspace replacement and workspace membership. Its old `go.work`
  directive reproduced Go's rejection of the newer module requirement; the
  fixture now declares the selected toolchain version.

All focused checks listed above passed before review. In particular, the
OpenAPI contract suite and external-security specimen passed with the retained
Vacuum snapshot and libopenapi v0.42.5. The full SQLite race suite took 20.419s
on macOS and 23.570s on Linux; PostgreSQL acceptance took 21.646s.

Limit build concurrency on memory-constrained hosts (`GOMAXPROCS=2`,
`GOFLAGS=-p=2`). Overlapping cold builds exhausted the initial contract timeout
and the Linux example-build deadline; those attempts are not passing evidence.
A complete importer race run also exhausted a 20-minute command limit while
progressing through generated cases, without an assertion failure. It is not
counted as a pass; the focused race checks establish the changed behavior, and
the normal full suite runs with the repository's standard 40-minute limit.
No dependency source, contract rule, timeout or test assertion was changed to
accommodate the update.
