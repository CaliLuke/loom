# Test Suite Audit — loom

Audited: 2026-07-11 · Commit: 90316cac · Auditor: test-suite-audit skill

## 1. Executive summary

The suite is 286 test files / 945 test functions plus 1,305 golden files, and it is fast and healthy by most measures: the full unit suite runs in ~25–35 s wall locally, integration in ~58 s, 69.6% statement coverage, and 17 of 20 sampled bug-fix commits shipped with tests. **Value** is good at the core (disciplined table-driven golden tests, no tautologies found by repo-wide grep) but one test is permanently dead (an env-gated test no environment ever enables — V-2; an initially reported second dead test, V-1, turned out to be false on runtime verification), and ~550 implementation-coupled substring assertions duplicate what goldens already pin. **Completeness** is the weakest dimension: the coverage is inverted relative to risk — runtime packages users depend on (`grpc` runtime 39.6%, `clue/log` 12.2%, `pulse/pool` 21%, `dsl` direct 38.7% with the repo's highest churn) sit far below the codegen packages (78–88%), and the repo's own CLAUDE.md-admitted gap (the GET `events/stream` SSE listener) remains behaviorally untested. **Speed** is dominated by a tiny tail: the slowest 45 tests (5%) consume 97.4% of the 50.7 s summed unit-test time, almost all of it subprocess builds (`go test` of generated modules, `protoc`, `npx redocly`, real-binary builds), and the integration tree performs ~20 serial Go compilations per run. The single most important number: **local `make all` is ~2.3 min today and ~1 min is achievable** by amortizing subprocess builds and hoisting per-test generation to per-session; separately, Windows CI (the 455 s pipeline critical path) silently executes zero lint and zero integration tests.

## 2. Inventory (cross-repo baseline metrics)

| Metric | Value |
|---|---|
| Languages / frameworks | Go 1.26 (single language); design-first codegen framework (Goa fork) |
| Test runner(s) | `go test` via `make test` / `make integration-test`; testify (`require`/`assert`); golden helper `codegen/testutil` with `-update`/`-u` flags; GitHub Actions (ubuntu + windows matrix) |
| Test files / test cases | 286 `_test.go` files; 945 top-level `Test` funcs (916 root module + 29 in 2 integration modules); 0 benchmarks/fuzz/examples; 911 top-level tests execute in a root-module run |
| Layer split (unit / integration / e2e) | 916 unit-layer funcs (incl. codegen golden + compile-smoke tests); 29 integration funcs driving generated apps over real TCP (+78 YAML scenarios via the jsonrpc framework); no separate e2e layer (integration is the e2e layer) |
| Total wall time (local) / (CI pipeline) | Unit: 24.4 s cold, 26.5 s / 35.3 s warm (±33% run-to-run variance); `make test` with coverage 63.9 s; integration 58.2 s; lint 15.4 s. CI (last green run): ubuntu `make ci` job 261 s, windows job 455 s (pipeline critical path), openapi-contract 45 s, generated-code-quality 121 s |
| Parallelism today | Package-level (GOMAXPROCS) only; 80 `t.Parallel()` call sites across 286 files; core utilization of a warm run ≈ 1.9 of 10 cores (user+sys 51.5 s / wall 26.5 s); no CI sharding |
| Coverage (line / branch) | 69.6% statements (`go test --coverprofile`, the repo's own documented path, works). Branch coverage: not measurable (Go tooling has no branch mode) |
| Skipped/disabled tests | 4 `t.Skip` sites; **1 permanently dead** (V-2); 1 unreachable-in-practice guard (V-1, corrected); 1 documented-intentional (`eval/error_location_test.go:15`), 1 conditional (`http/codegen/websocket_golden_test.go:94`) |
| Snapshot tests (count / size on disk) | 1,305 `.golden` files / 5.4 MB (testdata overall: 2,036 files / 8.9 MB; largest: `http/codegen/testdata` 5.8 MB, 739 goldens) |
| Sleeps in tests (count / summed literal seconds) | 2 literal sleeps ≈ 0.26 s total: 10 ms bounded ordering primitive (`grpc/middleware/canceler_test.go:34`), 250 ms injected into generated source by `jsonrpc/integration_tests/tests/sse_headers_test.go:65` |
| Bug-fix commits with tests (sampled ratio) | 17/20 (85%) of bug-fix commits since 2026-01 include test changes (untested: 82e4c552, a4e3452c, plus 3f3c1028 test-light) |
| Sampling performed | 5 subagents read ~60 of 286 test files in depth across 6 clusters; all literal-pattern claims verified by repo-wide greps, not sampling; ≥1 citation per subagent spot-checked in main context (all passed) |

Suite provenance: repo dates to 2015 (Goa upstream, 3,468 commits); the Loom fork's own activity is 2026. The repo's own planning docs record known gaps: `roadmap/refactor_checklist.md` has **108 open / 0 done** checklist items (including "add characterization coverage before restructuring"), `roadmap/finish_checklist.md` 31 open / 10 done, and `CLAUDE.md` itself documents the SSE GET-listener coverage hole (see C-1).

## 3. Findings

### V-1 (CORRECTED — original finding was false): `TestStreamingErrorComparison` runs; its silent-skip guard was fragile, not dead
- **Severity:** low (originally misreported as high)
- **Confidence:** measured
- **Evidence:** The original finding claimed the test always skips because `"client-stream-recv"` appears nowhere in non-test source. The literal grep was correct but the inference was wrong: the section name is built dynamically (`grpc/codegen/jennifer_stream.go:84`, `stream.Type+"-stream-recv"`), and the recorded `go test -json` run shows `TestStreamingErrorComparison` **passed** (not skipped) — its assertions do execute. The real (minor) defect was the guard at `streaming_errors_test.go:176–178`: had the sections ever gone missing, the test would silently skip instead of failing.
- **Impact:** None today. The lesson is recorded for audit method: a grep-negative on a section-name literal cannot establish dead code when names are constructed dynamically; runtime skip events are the authoritative source.
- **Recommendation (applied):** The silent skip has been replaced with `require.NotEmpty` on both section lookups so a future rename fails loudly.

### V-2: `TestJSONRPCSSEIntegration` never runs in any environment
- **Severity:** high
- **Confidence:** measured
- **Evidence:** `jsonrpc/codegen/sse_integration_test.go:15–17` skips unless `LOOM_INTEGRATION_TEST` is set. `grep -rn LOOM_INTEGRATION_TEST` across `.github/`, `Makefile`, and `scripts/` returns 0 hits; `make integration-test` never enters `jsonrpc/codegen`.
- **Impact:** The cluster's only would-be behavioral SSE test is dead weight and a selection hole — readers assume SSE has integration coverage here. Compounds C-1.
- **Recommendation:** Wire the env var into a Makefile target/CI job, convert to a `//go:build integration` tag so dormancy is explicit, or delete and rely on the integration trees.

### V-3: ~550 implementation-coupled substring assertions duplicate the golden layer's brittleness
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `Contains`/`NotContains` on generated source: 258 in `jsonrpc/codegen` (vs only 4 golden files there), 212 in `http/codegen`, 83 in `codegen/service`. Whole-line examples: `jsonrpc/codegen/sse_test.go:203,239,318–321` (exact whitespace-sensitive blocks), `http/codegen/sse_server_test.go:60–72` (12 exact substrings incl. comment text), `render_sections_test.go:60–62`, `codegen/service/service_dedup_test.go:32` (`strings.Count(...) == 1` on an exact receiver signature). `jsonrpc/codegen/sse_test.go` also repeats a ~12-line section-extraction loop 5× (`:296–380`) although its own helper `fileSectionCode` (used at `:244`) already does this.
- **Impact:** Cosmetic emitter changes (spacing, renames, reorders) fail dozens of tests with no behavior change — refactor friction on the highest-churn packages, duplicating protection goldens already provide.
- **Recommendation:** Migrate the dense clusters (jsonrpc mount/handler sections, `sse_server_test.go`) to golden comparison via the existing `-update` mechanism; reserve `Contains` for genuinely load-bearing tokens (e.g. the ordering guard at `sse_server_test.go:93–97`). Route `sse_test.go` extraction through `fileSectionCode`.

### V-4: Golden governance gaps — bulk regens of 500–700 files and self-seeding goldens
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** Commit `8931cbbf` touched 1,463 files (705 goldens, +52,017 lines); `7df1ed4a` 545 goldens; `a7433405` 180 goldens in one behavior-fix commit. Self-seeding: `http/codegen/websocket_golden_test.go:102–110` writes a missing golden and passes without `-update`, unlike the safe helper (`codegen/testutil/golden.go:118–119` fails with "run with -update to create"). Counterpoint: targeted commits in `codegen/*` and `grpc/codegen` history pair golden churn with real source changes — no evidence of routine blind `-update` sweeps outside the big rename/baseline commits.
- **Impact:** A mechanical `-update` across hundreds of files can bake a regression into the baseline unreviewed; the websocket path lets a wrong first value become "correct" silently.
- **Recommendation:** Make the websocket missing-golden case fail like `golden.go` does. Consider a CI guard flagging PRs that change generator templates and regenerate >N goldens with no accompanying assertion change.

### V-5: The same SSE behaviors are pinned at three layers, and fixture tests are near-copies across the two integration trees
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** SSE header-commit/pre-stream-failure behavior is asserted (1) as substrings in `http/codegen/sse_server_test.go:45–99`, (2) behaviorally in `http/integration_tests/tests/sse_fixture_test.go` + `sse_adversarial_test.go`, and (3) fixture freshness again via `scripts/generated_code_quality.sh:30–34` (a third regeneration of the same ticktock/mixedtick fixtures in a separate CI job). `sse_client_close_test.go` exists in both trees' ticktock fixtures with 3 identically-shaped tests (the http copy has 4 extra cases — already drifting); `Test*SSEFixtureRegeneratesAndBuilds` is structurally duplicated across `jsonrpc/.../sse_adversarial_test.go:82–96` and `http/.../sse_adversarial_test.go:50–64`.
- **Impact:** A framing change forces edits in ≥3 places; the drifting near-copies multiply maintenance without adding distinct defect detection.
- **Recommendation:** Keep the compiled integration fixture as the behavioral source of truth; slim `sse_server_test.go` to golden + ordering checks (see V-3). Factor the shared close/regen harness into `internal/testingx`. Decide whether fixture-freshness belongs to `generated-code-quality` or the integration test — not both.

### V-6: Assertion-light "produced something" tests concentrated in three service-codegen files
- **Severity:** low
- **Confidence:** measured
- **Evidence:** `len>0`/`NotEmpty`/`NotNil`-style assertions: `codegen/service/service_data_refactor_test.go` (29), `security_test.go` (20), `transport_descriptors_test.go` (19). Also `http/codegen/websocket_golden_test.go:119–152` `TestWebSocketTemplateExercise` asserts only render-without-error, overlapping the 24 golden cases. Repo-wide greps for true tautologies (`require.True(t, true)` etc.) and commented-out tests: 0 hits.
- **Impact:** These pass under substantial regressions; they are guards, not tests.
- **Recommendation:** Pair each with a shape/value assertion or fold into neighboring golden tests; drop `TestWebSocketTemplateExercise`.

### V-7: Unused test-infrastructure API surface in `codegen/testutil`
- **Severity:** low
- **Confidence:** measured
- **Evidence:** Zero references outside the package for `NewGoldenFile`, `SetUpdateMode`, `DefaultBasePath`, `TestField`, and the `ProjectionParity*`/`FormatProjectionParityDiffs` primitives (only self-tested; real consumers use package-level `Assert`/`AssertGo` — 54/45 uses — and `AssertProjectionParity` has exactly one caller, `codegen/service/views_test.go:105–107`). Explains the package's 61.3% coverage.
- **Impact:** Dead API maintained inside the most-depended-on test helper.
- **Recommendation:** Delete the unused `GoldenFile` object API or migrate a consumer to it.

### V-8: Stray untracked scratch file compiled by the integration module
- **Severity:** low
- **Confidence:** measured
- **Evidence:** `jsonrpc/integration_tests/codex_tmp_repro_main.go` (705 bytes, `package main`, dated Jul 10) sits at the module root, untracked, and is compiled by any `go build ./...`/`go test ./...` in that module.
- **Impact:** Debug residue; API drift in `framework` would break the module build for whoever has it locally.
- **Recommendation:** Delete it.

### C-1: The GET `events/stream` SSE listener contract has zero behavioral coverage — the repo's own docs admit it
- **Severity:** high
- **Confidence:** measured
- **Evidence:** `CLAUDE.md` ("the checked-in JSON-RPC ticktock fixture only proves POST-initiated SSE behavior; it does **not** cover the raw `events/stream` GET listener contract") and `jsonrpc/integration_tests/.../SCOPE.md:31–36` say the same. Generated servers contain the branch (`fixtures/ticktock/gen/jsonrpc/clock/server/server.go:196,258`; mixedtick `server.go:506`), but every integration test initiates via POST (`sse_external_client_test.go:69`, `sse_adversarial_test.go:41`, `sse_mixed_transport_test.go:74,149`; grep for `MethodGet` in jsonrpc test code: 0). Codegen-side coverage is string-only (`jsonrpc/codegen/sse_test.go:177,207,296,356` assert substrings of emitted source); the one compile-and-run smoke (`loomhttp_alias_regression_test.go:44`) drives the POST DSL. The only behavioral test for this branch is V-2's never-running test.
- **Impact:** An entire generated server branch (GET-initiated SSE: open-before-first-frame, origin protection, framing) can be string-correct but behaviorally broken and still pass everything. Domain criticality: this is the transport contract the framework exists to guarantee.
- **Recommendation:** Add a fixture test issuing a real `GET .../events/stream` with `Accept: text/event-stream`, asserting framing and lifecycle — per CLAUDE.md's own instruction. This is the top completeness item.

### C-2: `dsl` — highest-churn package, 38.7% direct coverage; 74 of 175 exported DSL functions have zero direct test references; the "spec" tests assert nothing
- **Severity:** high
- **Confidence:** measured
- **Evidence:** 64 commits/12 months (161 file-changes — #7 by churn), 2,044 stmts at 38.7%. Zero in-package test references for whole clusters: security (`APIKeySecurity`, `JWTSecurity`, `OAuth2Security`, all four OAuth flows, `Scope`, …; `dsl/security.go` is 2.6% covered / 189 stmts), SSE (`ServerSentEvents`, `SSEEventData/ID/Retry/Type`, …), CORS (`CORS`, `Origin`, `OriginRegex`, …), streaming (`StreamingPayload`, `StreamingResult`), `ConvertTo`/`CreateFrom`, and 7 HTTP verb functions. `dsl/_spec/dsl_spec_test.go` + `type_spec_test.go` contain 0 `func Test` — package-scope `var _ = API(...)` compile-eval smoke only (verified: `grep -c "func Test"` = 0). Mitigation: 60 external test files import `loom/dsl`, so real protection is higher than 38.7% — but only incidental, via codegen goldens.
- **Impact:** Validation/eval regressions in security, CORS, and SSE DSL functions (auth-domain criticality) surface only if they happen to change a downstream golden byte.
- **Recommendation:** Add direct eval-and-assert tests for the security/SSE/CORS/streaming clusters (valid design → expected expr fields; invalid design → expected eval error). Convert `_spec` into asserting tests. `roadmap/refactor_checklist.md`'s own (all-open) items call for exactly this characterization coverage.

### C-3: `grpc` runtime — client invoker and unary/stream handlers entirely untested (39.6%)
- **Severity:** high
- **Confidence:** measured
- **Evidence:** Only `grpc/error_test.go` and `grpc/protobuf_value_test.go` exist. Zero test references for `NewInvoker`/`Invoke` (client.go), `NewStreamHandler`/`NewUnaryHandler`/`Handle`/`Decode` (handler.go), `NewStatusError`, `ErrInvalidType` (verified grep: 0 hits outside codegen/middleware).
- **Impact:** The runtime dispatch/encode/invoke path every generated gRPC service runs through has no in-package tests; only fixture compile smoke touches it indirectly.
- **Recommendation:** Add `handler_test.go`/`client_test.go` over `bufconn` — highest-value runtime tests to write.

### C-4: `clue/log` — one test function for 1,716 lines (12.2%)
- **Severity:** high
- **Confidence:** measured
- **Evidence:** `clue/log/log_test.go` has 1 `func Test` (verified). Untested: `grpc.go` (278 lines), `http.go` (263), `format.go` (254, 1% covered), `adapt.go` (227, 0%), `options.go` (185). Sibling gaps: `clue/debug` 20.8%, `clue/mock` 0% (a test helper whose own sequence-dispatch logic is untested), `http/middleware/debug.go` 0%/110 stmts, `grpc/middleware/log.go` 0%/150 stmts.
- **Impact:** Largest untested surface by raw line count; logging interceptors/formatters fail silently. Churn is low (3 commits/12 mo), which tempers urgency.
- **Recommendation:** Test `format.go` first (pure, easy), then the HTTP/gRPC interceptors with `httptest`/bufconn.

### C-5: `pulse` — Redis-backed worker/scheduler/node runtime untested beyond targeted race-regression pins (pool 21%, rmap 17.8%, streaming 33%)
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** Unit tests cover pure helpers only (hand-built `redis.XMessage` structs; no miniredis/real Redis anywhere). Untested files: `pulse/pool/worker.go` (476 stmts @7.1%), `node_cleanup.go` (542 @4.1%), `ticker.go` (216 @5.6%), `node.go` (176 @0%), `pulse/rmap/map_runtime.go` (354 @0%), `map_mutation.go` (264 @0%). Mitigation verified: the recent race-fix commits **did** add tests — `0a3fdd17` added `pool_shutdown_regression_test.go` (+327 lines), `6e68d85f` added +204 to `streaming_test.go`.
- **Impact:** The distributed dispatch loops where races were just fixed are guarded only by those specific regression pins; new races/lifecycle bugs in adjacent paths have no net. One sampled untested fix: `a4e3452c` "Fix pulse stream option parsing" shipped with 0 test changes.
- **Recommendation:** Add miniredis-backed tests for worker/scheduler dispatch and rmap runtime. If pulse is not yet load-bearing for Loom users, say so in the roadmap and accept the risk explicitly.

### C-6: `jsonrpc/codegen/decoder_sections.go` — 29.1% covered with 15 changes in 12 months (worst churn×coverage cell in codegen)
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** Per-file coverage from `cover.out`: 227 stmts, 29.1% — against a cluster average of 84.7%. Sibling high-churn files are 88–99% (e.g. `top_level_sections.go` 99.6%). Whole-suite context: all other top-20 churn files measure 70–100%.
- **Impact:** Request-decoding codegen (malformed-input handling — the security-relevant edge of JSON-RPC) is the least-protected hot file in the transport layer.
- **Recommendation:** Add golden/table tests for the decoder sections' branches (optional params, invalid types, batch edge cases) before further decoder work.

### C-7: Middleware utility surface untested — `middleware` 40.7%, `http/middleware` 51.2%
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** No test file exists for `middleware/log.go` (`NewLogger`, `WrapLogger`, `WithSpan`) or `middleware/requestid.go` (`GenerateRequestID` + 5 option funcs). In `http/middleware`, zero references for `Log`/`LogContext`, `Debug`, `CaptureResponse` and `ResponseCapture.{Write,WriteHeader,Flush,Hijack,Push}` (interface-forwarding wiring that breaks silently), `PopulateRequestContext`, `WrapDoer`.
- **Impact:** Request-ID generation and response-capture forwarding are the kind of plumbing whose breakage corrupts observability without failing anything.
- **Recommendation:** `capture_test.go` (interface forwarding) and `requestid_test.go` first; both are cheap table tests.

### C-8: Codegen error/panic branches unexercised in hot generator files
- **Severity:** medium
- **Confidence:** inferred (statically identified branches; coverage corroborates)
- **Evidence:** No test trips `panic(codegen.NewError(...))` at `http/codegen/websocket.go:187` or `http/codegen/service_data_payload.go:447`, the 2 panics + 1 error return in `service_data_response.go`, or `codegen/validation.go:177` (`panic("unknown format") // bug`). `grpc/codegen/protobuf_transform.go` has 11 error sites; no test makes `ProtoBufTransform` return an error (the only error-adjacent assert checks a generated string, `protobuf_transform_test.go:231`). Positive counterexamples exist (`codegen/go_transform_test.go:199` and `go_transform_union_test.go:47` do assert real errors), proving the pattern is practical here.
- **Impact:** Invalid-design failure modes can panic with wrong messages or succeed wrongly; only happy-path golden bytes are pinned.
- **Recommendation:** Add negative-DSL table cases per file following the `go_transform_test.go` pattern.

### C-9: Windows CI silently executes zero lint and zero integration tests while being the pipeline's critical path
- **Severity:** medium (completeness) — see also S-7 (speed)
- **Confidence:** measured
- **Evidence:** `Makefile` guards `lint` (`:99`) and the entire `integration-test` body (`:125–129`) with `ifneq ($(GOOS),windows)` and no else/echo; `.github/workflows/test.yml` runs `make ci` on `windows-latest`, which expands those targets to no-ops and exits 0. The windows job took 455 s in the latest green run — the longest job in the pipeline — to run only unit tests.
- **Impact:** Windows CI green ≠ Windows works: the transport/exec/process-heavy integration surface (the code most likely to have OS-specific bugs — the 2024 upstream flake ledger entry d288ca34 was exactly a Windows socket issue) is unverified. Meanwhile the job costs the most wall time.
- **Recommendation:** Either make the skip loud (`@echo "SKIPPED on windows"`) and consciously accept unit-only Windows coverage, or port the integration harness (it already uses `net.Listen`/`GOWORK=off`, all cross-platform). Don't leave it implicit.

### C-10: Concurrency paths in `http/sse.go` and `http/websocket.go` lack race-exercising tests
- **Severity:** low
- **Confidence:** inferred
- **Evidence:** `http/websocket.go:35–37` (`connLock`, `writeLock`), `http/sse.go:117–135` (goroutine + double-select cancellation drain) have no concurrent-stress tests; the only explicit race test in the runtime clusters is `http/middleware/xray/segment_test.go:198`. The `readChunk` cancel-with-failed-Close branch is a plausible untested edge. Note the suite does not run `-race` in CI (`make test` has no race flag).
- **Impact:** Races in stream teardown would reach users first.
- **Recommendation:** Add a `-race` CI variant (ubuntu only, unit suite — it's a 25 s suite, cost is small) plus one concurrent-write websocket test and one SSE cancel-mid-read test.

### S-1: 97.4% of unit-suite time sits in 45 tests; the top 7 (≈42 s of 50.7 s) are all subprocess builds
- **Severity:** high
- **Confidence:** measured
- **Evidence:** Per-test timings from `go test -json` (warm): `TestRepresentativeSpecsPassRedoclyLintAndConsumerSmoke` 10.5 s, `TestFormRequestUnionOAuthIntegration` 6.85 s, `TestLargeErrorSetHTTPClientIntegration` 5.28 s, `TestProtoFiles` 5.20 s, `TestGenerateRealBinaryFailurePreservesStageContext` 5.19 s, `TestGenerateRealBinaryDebugContract` 4.91 s, `TestProtoc` 3.93 s. Sum of all 911 tests: 50.7 s; slowest 5% = 97.4%.
- **Impact:** The entire optimization surface is 7 tests; the other ~900 tests cost ~1.3 s combined.
- **Recommendation:** Work findings S-2 through S-5; nothing else in the unit suite is worth touching for speed.

### S-2: Network-dependent `npx` contract test runs unguarded in the default unit suite, and again in a dedicated CI job
- **Severity:** high
- **Confidence:** measured
- **Evidence:** `http/codegen/openapi/v3/contract_smoke_test.go:138,152,156` shells `npx --yes @redocly/cli@2.24.1` / `openapi-typescript` / `tsc` with no skip guard, no `exec.LookPath` check, no build tag (verified grep). It runs in every `go test ./...` (10.5 s measured) **and** again in the separate `openapi-contract` CI job (`Makefile` `openapi-contract` target selects the same tests by `-run` regex; job measured at 45 s).
- **Impact:** 10.5 s per local unit run; hard dependency on Node + npm registry inside the unit suite (offline/no-node machines fail `make test`); duplicate CI execution.
- **Recommendation:** Gate it behind `testing.Short()` or a build tag and let the `openapi-contract` job own it exclusively. Single change, –10.5 s from every local unit run.

### S-3: Two http/codegen tests each assemble and `go test` a full temp module — nothing shared, possible git fetch
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `http/codegen/form_union_oauth_integration_test.go:19–46`: both tests call `renderHTTPModule` → `checkoutPinnedLoomModule` (`:83–117`, which absent `LOOM_REPO`/`.loom_source_mode` does `git init` + shallow `git fetch` — network I/O) then `go mod tidy` + `go test ./...` in separate `t.TempDir()`s. 6.85 s + 5.28 s measured; neither uses `t.Parallel()`.
- **Impact:** ~12.1 s, dominated by two full module compiles of loom-as-dependency.
- **Recommendation:** Share one temp module (two packages), or run both harnesses under one `go test` invocation; ensure CI sets the source mode so the git-fetch path never runs.

### S-4: `cmd/loom` real-binary tests build the same generator binary twice and `os.Chdir` blocks parallelism
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `cmd/loom/generate_test.go:211` and `:260` both `os.Chdir` into `jsonrpc/integration_tests/fixtures/ticktock` and invoke `generate()` → real `go build` of the generator binary (`cmd/loom/gen.go:184`) + exec. 5.19 s + 4.91 s measured; the second test diverges only after binary startup.
- **Impact:** ~10.1 s; ~half is a duplicate build.
- **Recommendation:** Build once (`TestMain`/`sync.Once`), run twice with different output args; replace `os.Chdir` with a dir parameter to unlock `t.Parallel`.

### S-5: 24 serial `protoc` spawns + a fresh `go build` of the fake protoc every run
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `grpc/codegen/proto_test.go`: `TestProtoFiles` (14 cases, one `protoc` subprocess each at `:53`), `TestMessageDefSection` (10 more at `:87`), all serial (0 `t.Parallel` in the cluster); `TestProtoc:119` runs `go build ./testdata/protoc` on every invocation. 5.20 s + 3.93 s measured.
- **Impact:** ~9 s of process-spawn overhead.
- **Recommendation:** `t.Parallel()` in the per-case `t.Run`s (each already uses isolated temp files); build the fake protoc once via `sync.Once`.

### S-6: Integration tree performs ~20 serial Go compilations per run, ignores the prebuilt loom binary, and regenerates mixedtick three times
- **Severity:** high
- **Confidence:** measured
- **Evidence:** `jsonrpc/integration_tests/tests` measured at 50.3 s of the 58.2 s integration wall. Servers start via `go mod tidy` + `go run .` per test (`harness/server.go:79,112`); generation runs `go run -mod=mod github.com/CaliLuke/loom/cmd/loom` (`framework/generator.go:445,450`, `sse_adversarial_test.go:90`, `sse_mixed_transport_test.go:128`) even though `make integration-test` depends on `build-loom` which already installed the binary to PATH. `startMixedTickServer` (`sse_mixed_transport_test.go:120–138`) does CopyTree + `loom gen` + `go run` **per test**, called by all 3 mixed tests. Per-test measured: `TestSSEHeadersAreDeferredUntilFirstEvent` 27.6 s (the slowest test in the repo), `TestJSONRPCSSEFixtureRegeneratesAndBuilds` 17.1 s, three mixedtick tests ~14 s each (they overlap via `t.Parallel`).
- **Impact:** ~20 toolchain compiles per run where ~8 would do; `make build-loom` is wasted work for these tests.
- **Recommendation:** Use the PATH-installed `loom` binary; hoist mixedtick generation to once-per-package (`TestMain`); consider `go build` once + reuse server binary per fixture.

### S-7: CI structure — Windows critical path, duplicate executions, no suite sharding
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** Latest green pipeline: windows job 455 s (123 s Go setup + 320 s build/test for unit-tests-only — see C-9), ubuntu 261 s, generated-code-quality 121 s (51 s of it `make depend` reinstalling tools), openapi-contract 45 s. Duplicates: the openapi contract tests run in both the unit suite and their dedicated job (S-2); unit tests run on both OSes (justified as cross-platform coverage); fixture regeneration runs in both `generated-code-quality` and the integration tests (V-5). Local pre-push hook (`.githooks/pre-push` → `check.sh` = lint + `make test`) measured ≈ 79 s (15.4 s + 63.9 s) — under the friction threshold, acceptable.
- **Impact:** Pipeline wall time is set by the job that verifies the least.
- **Recommendation:** Cache Go build/module dirs on windows (biggest single CI win); remove the unit-suite copy of the npx tests (S-2); pick one owner for fixture freshness (V-5).

### S-8: Near-zero intra-package parallelism — 1.9 of 10 cores used
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** 80 `t.Parallel()` sites across 286 files; 4 of 51 top-level `http/codegen` files, 0 in `grpc/codegen` + `jsonrpc/codegen` (37 files), 0 in `cmd/loom`; the 200-case `TestDecode` table (`http/codegen/server_decode_test.go:13`) runs serially. Warm-run utilization: user+sys 51.5 s over 26.5 s wall ≈ 1.9 cores. Codegen tests are pure and independent (fresh DSL root per case) — trivially parallelizable, with one caveat: shared globals like `openapi.Definitions` reset (`auth_error_responses_test.go:17`) must be audited first.
- **Impact:** The ceiling is written in the number: even with S-2..S-5 fixed, serial execution leaves most of 10 cores idle inside the heavy packages.
- **Recommendation:** Add `t.Parallel()` to the big golden tables and the protoc/module-compile tests after S-3/S-4 remove their shared-state blockers.

### S-9: Order-coupling blockers: OTel global mutation (confirmed shuffle failure) and fixed X-Ray UDP ports
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `go test -shuffle=12345 ./...` passed; `-shuffle=98765` **failed**: `observability/otel/runtime_test.go:43` (`TestNewInitializesEnabledProvidersAndGlobals` mutates the global tracer/meter/logger providers + propagator via `New()` with no defer-restore; `TestNewDisabledSectionsDoNotReplaceUnrelatedGlobals:47` reads leaked state). Fixed UDP ports: `middleware/xray/segment_test.go:14` (`127.0.0.1:62111`), `http/middleware/xray/middleware_test.go:19` (`:62112`), `grpc/middleware/xray/middleware_test.go:64` (`:62113`) — port 62112 is the exact port in the 2024 upstream Windows-flake commit d288ca34; fixed ports forbid `t.Parallel` and multi-instance runs. The good pattern already exists in-tree: `http/middleware/xray/wrap_doer_test.go:51` binds `Port: 0`. The three xray suites are otherwise justified layering, not copies (base marshalling tested once in `middleware/xray`), and cost ~2.5–2.9 s each on 100 ms UDP read deadlines.
- **Impact:** One confirmed order-dependent failure; three packages structurally unparallelizable; a historical flake source still armed.
- **Recommendation:** Snapshot/restore the four OTel globals in every test calling `New()`. Switch xray suites to dynamic ports like `wrap_doer_test.go`. Then `-shuffle=on` can be added to CI as a standing guard.

### S-10: Test-run hygiene — 1.6 GB of leftover artifacts and in-place mutation of checked-in fixtures
- **Severity:** medium
- **Confidence:** measured
- **Evidence:** `du -sh jsonrpc/integration_tests` = 1.6 GB: 72 leftover `loomNNNN/loom` binaries (~23 MB each, gitignored, never cleaned) + ~30 `server-*.log` files. Tests point `StartServer` at the checked-in `../fixtures/ticktock` (`sse_external_client_test.go:25–26`, http `sse_fixture_test.go:23–24`), writing logs into it and running `go mod tidy` in place (mutating tracked `go.sum`); `RequireTreeMatches` then compares against a fixture other tests may have mutated.
- **Impact:** Unbounded disk growth; fixture-pristineness assumptions silently violated; potential false regen-diff failures.
- **Recommendation:** Run in-place fixture tests on a `t.TempDir()` CopyTree (the mixed tests already do); `t.Cleanup` the logs/binaries; add a `make clean` for the gitignored cruft.

### Fixture-architecture verdict (required): **keep, improve governance.** The golden + checked-in-fixture strategy is economically sound — goldens cost nothing at runtime, regeneration is a real `-update` flag, and history shows disciplined targeted regens. The governance gaps are V-4 (self-seeding, bulk-regen review risk), V-5 (three owners for the same fixtures), and S-10 (in-place mutation) — fix those; do not restructure.

### Flake verdict (required): no flake reproduced in 6 canonical-order runs plus 1 run under subagent load (idle-to-moderate machine, macOS/10 cores). One **deterministic order-coupling** confirmed under shuffle seed 98765 (S-9). The flake ledger (7 commits, latest 2024, all upstream-era) points at the xray UDP fixed ports still present today. Timeout-proximity: no test exceeds 50% of its budget (worst: integration package 50.3 s vs 600 s). The 250 ms-vs-100 ms timing race in `sse_headers_test.go` (V-5/S-6 file) is the most likely future flake; it passed in all observed runs.

## 4. Checklist coverage

Value (value.md):
- 1.1 tautological tests — **not found** (repo-wide greps for literal tautologies: 0; assert-nothing: found only `dsl/_spec` compile-smoke (C-2) and render-only `TestWebSocketTemplateExercise` (V-6))
- 1.1 broad-catch swallowing — **not found** (13 `recover()` in tests are all panic-assertion helpers)
- 1.2 over-mocked/wiring tests — **not found** (no mock framework in use; hand-rolled fakes minimal; the weak-assertion family is V-6)
- 1.2 stub-echo/hardcoded-message tests — **not found**
- 1.3 near-duplicates/parametrization candidates — **found, mostly already table-driven** (verified `server_decode_test.go` 200 cases/200 goldens/0 orphans); residual: `sse_test.go` boilerplate (V-3), xray `TestRecordError` duplicated base+http (minor)
- 1.3 same seam from multiple roots — **found** (SSE tri-layer + cross-tree copies, V-5)
- 1.3 copy-pasted files (md5 pass) — **not found** (0 duplicate hashes)
- 1.4 snapshot/golden audit — **found** (1,305 goldens/5.4 MB; bulk regens + self-seeding, V-4; 25 websocket goldens >100 lines, largest 195)
- 1.5 testing someone else's code — **not found** in samples
- 1.6a test roots — **enumerated**: root module (in-package tests), `dsl/_spec`, 2 integration modules, fixture-embedded tests (`fixtures/*/sse_client_close_test.go`), `grpc/integration_tests/fixtures/quality` (design-only, exercised solely by `generated_code_quality.sh` — no go test selects it). Selection holes: V-2 (env-gated, never selected), C-9 (windows selects nothing for lint/integration)
- 1.6 dead weight — **found**: 1 permanently dead test (V-2; the initially reported V-1 was corrected on runtime evidence), 1 stray scratch file (V-8), 0 commented-out tests; duplicate CI executions (S-7)

Completeness (completeness.md):
- 2.1 criticality ranking — **done** (churn table §2; domain-critical: dsl/security, jsonrpc decoder, transport runtimes)
- 2.2 repo's own coverage path — **works** (`make test` writes cover.out; CI uploads it as an artifact; nothing gates on it — noted, low)
- 2.2 per-directory coverage × churn — **done** (§2 + C-2..C-7; worst cells: dsl 38.7%@high churn, decoder_sections.go 29.1%@15 changes, pulse/clue at 12–33%@low churn)
- 2.2 zero-test-file source dirs — **found**: `security/` (12 stmts), `clue/mock`, `middleware/log|requestid` (C-7); `grpc/pb` generated (OK)
- 2.2 attribution integrity — **checked**: `codegen/template` and `codegen/codegentest` 0% are cross-package measurement artifacts, not gaps (verified consumers); dsl 38.7% under-states real protection (60 external importers) — both caveated
- 2.3 top-churn × test-reference cross-check — **done via per-file coverage** (all top-20 churn files 70–100% except `decoder_sections.go` 29.1%, C-6)
- 2.3 untested public surface — **found**: 74/175 dsl exports (C-2), grpc runtime exports (C-3), middleware exports (C-7), 5 codegen/cli exports (`UsageCommands`, `UsageExamples`, `CommandUsage`, `PayloadBuilderSection`, `NewFlagData` — covered only via distant http/grpc goldens)
- 2.4 error paths — **found** untested panic/error branches (C-8)
- 2.5 boundaries/edge inputs — **partially covered** by validation/table tests; no systematic gap beyond C-8 identified in samples
- 2.6 integration seams — **found**: pulse Redis runtime mock-only (C-5)
- 2.6 concurrency — **found**: sse/websocket thin (C-10); no `-race` in CI
- 2.6 migrations/config-flags/property-based — **N/A / not found / noted**: no migrations; property-based absent (opportunity for `expr` transform/validation domains, not a defect)
- 2.6 browser/e2e layer — **N/A** (no UI); integration trees are the e2e layer and exist
- 2.7 regression discipline — **measured**: 17/20 with tests; untested examples: 82e4c552 (WebSocket pre-upgrade error responses), a4e3452c (pulse option parsing)
- 2.8 mutation spot-check — **manual only**: `validation.go:177` unknown-format panic and `websocket.go:187` transform-panic mutants would survive (C-8); no mutation tooling configured

Speed (performance.md):
- 3.1 baseline/wall/CI/history — **measured** (§2; cold 24.4 s, warm 26.5/35.3 s, coverage 63.9 s, integration 58.2 s; CI history queried via `gh run list`/`gh run view`, last green step timings in S-7; note: 6 of 10 recent runs failed — all on CI-infra or in-flight jsonrpc work, since fixed)
- 3.1 distribution/top-slowest/5% share — **measured** (S-1: 97.4%)
- 3.1 overhead split — **measured**: build overhead ≈ cold−warm ≈ negligible with warm cache; coverage instrumentation adds ~2× (63.9 s vs ~30 s)
- 3.1 variance discipline — **observed** warm-run variance 26.5–35.3 s (±33%); all comparisons use warm runs; integration `-json` rerun overlapped subagent greps (flagged load-contaminated; used only for per-test split, not totals)
- 3.2 fixture/setup cost — **found**: per-test subprocess builds are the "fixtures" here (S-3..S-6)
- 3.2 sleeps — **measured**: 2 literals ≈ 0.26 s (§2) — not a time sink
- 3.2 real I/O in unit tests — **found**: npx/npm network (S-2), possible git fetch (S-3), real UDP sockets in xray (S-9)
- 3.2 retries hiding flakiness — **not found** (no retry annotations/plugins; bind-retry loop in xraytest noted)
- 3.2 lint jobs inside suite — **found**: the Redocly/tsc contract lint runs inside the unit suite (S-2)
- 3.2b flake hunt — **done** (verdict above; ledger counted, timeout-proximity ranked, 6-run identity diff, 1 loaded run)
- 3.2c fixture architecture verdict — **keep / improve-governance** (stated above with economics)
- 3.3 parallel today / core utilization — **measured** (S-8: 1.9/10 cores)
- 3.3 order-independence probe — **run, 2 seeds**: pass/fail split; failure named (S-9). Multi-process shard experiment: **not run** — `go test` already runs packages in parallel processes; the shuffle probe + utilization measurement answer the same question for Go's execution model
- 3.3 global mock/mutable state — **found**: OTel globals (S-9), `openapi.Definitions` global reset (S-8 caveat), 1 test file using `t.Setenv`, 4 `os.Chdir` sites (S-4 + generator tests)
- 3.3 shared external resources/fixed ports — **found**: 3 fixed UDP ports (S-9); integration servers use dynamic `:0` ports (clean)
- 3.4 CI structure — **found**: no caching on windows path, duplicate executions, no sharding (S-7); local hook latency measured 79 s (acceptable)
- 3.5 achievable target — **stated below**

## 5. Top opportunities (ranked)

| Rank | Opportunity | Findings | Type | Effort | Expected payoff |
|---|---|---|---|---|---|
| 1 | Gate the npx contract test out of the unit suite; let the openapi-contract job own it | S-2, S-7 | quick win | hours | −10.5 s every local unit run; removes network/Node dependency from `make test`; kills a CI duplicate |
| 2 | Fix the dead env-gated test and the fragile silent skip | V-1, V-2 | quick win | hours | Removes a false sense of SSE integration coverage; makes section renames fail loudly |
| 3 | Fix OTel global restore + xray dynamic ports, then add `-shuffle=on` (and `-race`) to CI | S-9, C-10 | quick win | hours–1 day | Unblocks parallelism in 4 packages; standing order-independence guard; race detection on a 25 s suite |
| 4 | Amortize subprocess builds: share the http/codegen temp module, build the cmd/loom binary once, cache the fake protoc, parallelize protoc cases | S-3, S-4, S-5, S-8 | quick win | 1–2 days | ~20 s off the unit suite's 50.7 s summed time; wall ~25–35 s → ~12–15 s |
| 5 | Integration tree: use the prebuilt loom binary, hoist mixedtick generation to per-package, stop mutating checked-in fixtures, clean artifacts | S-6, S-10 | structural | 2–4 days | 58 s → ~30 s; ends 1.6 GB disk growth and fixture-pristineness hazards |
| 6 | Write the GET `events/stream` behavioral fixture test | C-1, V-2 | structural | 1–2 days | Closes the repo's own top admitted gap; protects a whole generated branch |
| 7 | Direct DSL tests for security/SSE/CORS/streaming clusters; make `_spec` assert | C-2 | structural | 3–5 days | Direct protection for the highest-churn, auth-critical surface (74 uncovered exports) |
| 8 | Runtime package tests: grpc handler/client (bufconn), http/middleware capture/requestid, clue/log format | C-3, C-4, C-7 | structural | 1–2 weeks | Raises the floor under the code users actually run; grpc runtime first |
| 9 | Consolidate SSE pinning: goldens for jsonrpc string asserts, one owner for fixture freshness, shared close/regen harness | V-3, V-5 | structural | 3–5 days | ~550 brittle assertions → goldens; framing changes touch 1 place instead of 3 |
| 10 | Golden governance: fail on missing websocket golden; CI guard for bulk regens; make the Windows skip loud (or port integration) | V-4, C-9 | quick win + decision | hours (+days if porting) | Prevents silent baseline corruption; makes the Windows coverage gap an explicit decision |

## 6. Suggested refactoring sequence

Start with the quick wins that change what every developer feels daily: extract the npx test from the unit suite (rank 1) and amortize the subprocess builds (rank 4) — together they take the local unit loop from ~30 s to ~12–15 s, and nothing else depends on them. In the same pass, fix the dead env-gated test and the fragile silent skip (rank 2). Next do the isolation fixes (rank 3) *before* adding any `t.Parallel()` calls, because shuffle/race guards must be green before parallelism lands; once OTel and xray are clean, add `-shuffle=on` and `-race` to the ubuntu CI job and re-measure — expect the unit wall to drop further as the big tables parallelize. Then restructure the integration tree (rank 5); re-measure `make integration-test` after hoisting mixedtick (expect ~half). Only then start the completeness work, in risk order: the GET listener fixture (rank 6) unblocks deleting the dead env-gated test properly; DSL direct tests (rank 7) should precede any DSL refactoring — note `roadmap/refactor_checklist.md` already demands this and has 108 open items; runtime tests (rank 8) can proceed in parallel with anything. Leave the consolidation work (rank 9) for a quiet period — it's maintenance-cost reduction, not defect-risk reduction — and take the Windows decision (rank 10) explicitly with the team rather than letting the Makefile decide it silently.

**Achievable-target arithmetic (3.5):** Local `make all` today ≈ 15 s lint + 64 s test-with-coverage + 58 s integration ≈ **137 s**. After ranks 1+4 (−10.5 s npx, −~20 s amortized builds, remainder package-parallel): unit ≈ 12–15 s (+~25 s if coverage instrumentation retained — consider decoupling coverage from the default target). After rank 5 (mixedtick 3×→1×, prebuilt binary replacing ~10 `go run` compiles): integration ≈ 30 s. Achievable local `make all` ≈ 15 + 15 + 30 ≈ **60 s (~2.3×)**. CI critical path 455 s (windows) → ~180–250 s with Go build/module caching on windows plus the unit-suite npx removal; ubuntu 261 s → ~200 s. These estimates assume the measured warm-cache costs above; re-measure after each step.

## 6b. Post-audit fixes applied (same day)

Ranks 1–3 and 10 plus hygiene were applied immediately after the audit; measured after-numbers: `make test` 63.9 s → 37.7 s, unit-suite `openapi/v3` package 16.9 s → 2.9 s, `-shuffle` green on 3 seeds (incl. the previously failing 98765), `jsonrpc/integration_tests` 1.6 GB → 564 K. Specifics: S-2 (npx test env-gated via `LOOM_OPENAPI_CONTRACT`, set by `make openapi-contract`); V-2 (env gate removed — the test had rotted while dormant: its `"JSON-RPC notification"` client assertion no longer matched generated output, confirming the finding's impact; assertions refreshed); V-1-corrected (silent skip → `require.NotEmpty`); S-9 (OTel global snapshot/restore incl. `autok_compat_test.go`; xray suites use a per-binary dynamic UDP port via `xraytest.FreeUDPAddr` + `TestMain`); S-5 (protoc table tests: serial DSL phase, parallel compile phase); V-4 (websocket golden self-seeding removed); C-9 (Makefile prints loud SKIPPED lines for lint/integration-test on Windows); V-8/S-10-partial (scratch file deleted — it was tracked, not untracked as reported; `make clean` target added and run).

**Round 2 (same day):** the structural items landed as well.
- **C-1 closed**: `jsonrpc/integration_tests/tests/sse_get_listener_test.go` regenerates a temp events/stream variant of the ticktock fixture and drives a real `GET /rpc` with `Accept: text/event-stream`. It pinned a previously unspecified contract: streamed `Send` values arrive as JSON-RPC notifications, but the **`SendAndClose` final value is intentionally dropped** on the GET listener (no request ID → no final response, `jsonrpc/codegen/stream_sections.go` "Notifications are closed without a final response") and the server closes the stream. First drafted as "final value arrives," the test failed and exposed this semantic — now documented in the test, CLAUDE.md/AGENTS.md, and the fixture SCOPE.md. Costs ~16–20 s of integration wall (full temp codegen + module build), partly offset by S-6.
- **C-2/C-3/C-4/C-7 (subagent-written suites, all verified green with `-race` where applicable)**: `grpc` runtime 39.6% → 93.3% (client/handler via fakes + transport-stream stubs); `clue/log` 12.2% → **100%** (6 new/extended test files); `middleware` 40.7% → 83.1% and `http/middleware` → 94.4%; `dsl` direct 38.7% → 49.7% (security/SSE/CORS/streaming clusters, 30 invalid-design cases).
- **S-6**: mixedtick fixture generation hoisted to once-per-package (`sync.Once` + `TestMain` cleanup; server starts serialized on a mutex); the three mixed tests dropped from ~43 s of summed work to 11.6 s wall isolated. **S-3**: the pinned loom checkout in `http/codegen` is memoized (one git fetch per package run, `TestMain` cleanup). **S-4**: intentionally left — the two cmd/loom tests exercise the full `generate()` pipeline as their contract; further amortization would require production changes for test speed.
- **New order-coupling found and fixed**: `expr/method_test.go` detached-method test read API requirements off the leaked global `expr.Root` (failed under `-shuffle=777`); it now resets via `expr.SetupTestDSL`. Shuffle re-verified green on seeds 777 and 31337.
- **Production-bug suspicions surfaced by the new tests** (tests pin current behavior; no production code changed): `grpc.NewStatusError(codes.OK, …)` returns nil (error silently swallowed); `http/middleware/trace.go` `tracedDoer.Do` panics on a context with `TraceIDKey` but no span ID; `clue/log` `LogrSink.WithName` mutates the receiver (sibling sinks get cumulative names); `clue/log` `client.RoundTrip`+`WithLogBodyOnError` returns a consumed body on read failure; `responseCapture`/`ResponseCapture` log status 0 for implicit-200 handlers; `dsl/security_fields.go` `APIKey` has a mangled GoDoc; `SSENotificationMethod`'s JSON-RPC-only restriction is documented but not validated.
- Timing note: post-round-2 wall numbers (unit 42–66 s, integration ~90–170 s) were measured under heavy external machine load (loadavg 12–19 on 10 cores) and are not comparable to the morning baselines; the clean same-conditions comparisons are the isolated package numbers above. The integration suite genuinely gained ~16–20 s for the new GET-listener coverage.

**Round 3 (same day):** C-5 (pulse vs miniredis: rmap 17.8%→75.0%, streaming 25.8%→75.4%, pool 21%→~72%), C-6 (decoder_sections.go 29.1%→100%, package 84.7%→92.8%), C-8 (all five flagged panic/error sites covered; a latent swallowed-transform-error bug found — issue #148), C-10 (`make test-race` = `-race -shuffle=on` + a dedicated CI job; its first run caught real data races in the four `t.Parallel` http/codegen files, whose unsound parallelism was removed), and the V-3 `sse_test.go` consolidation (606→405 lines) all landed. The two hard bugs (`NewStatusError(codes.OK)` swallowing errors, `tracedDoer.Do` panic) are fixed; the remaining suspicions are filed as issues #142–#148, including the GET-listener `SendAndClose` semantics decision (#146).

Still open: S-7 CI caching, the V-5 cross-tree dedup remainder, and the issue backlog above. Decided later the same day: the V-3 golden migration landed (299 substring asserts → 149 + 76 goldens, no-lost-assertions gates 25/25, 55/55, 86/86), and Windows moved out of the per-PR pipeline to a weekly compile+unit canary (`windows-canary.yml`) — the audience is mac/linux and the 455 s Windows job only ran unit tests.

## 7. Method notes

Executed (macOS 15 / Darwin 25.5, Apple Silicon, 10 cores, Go per go.mod): 6 full unit-suite runs (1 cold, 2 warm timed, 1 coverage via the repo's documented `--coverprofile` path, 2 shuffled with seeds 12345/98765), 1 `go test -json` run for per-test timings, 1 timed `make integration-test` plus 1 `-json` rerun of the jsonrpc tests package (flagged: overlapped subagent grep load — used only for the per-test split), timed `make lint`. CI history queried via `gh run list`/`gh run view` (step-level durations from the latest green run). Coverage aggregated per-directory and per-file from `cover.out` and crossed with 12-month git churn. All value/performance checklist greps run repo-wide from the repo root over all test roots; hit counts recorded in §4. Not executed: mutation tooling (none configured; manual mutant analysis only), Redocly job internals, multi-process shard experiment (rationale in §4 — Go's per-package process parallelism plus the shuffle probe covers the order-independence question). Five Explore subagents audited clusters statically (instructed not to build/test to keep timing runs clean); ≥1 file:line citation per subagent was re-verified in the main context (`generate_test.go:211/260`, `streaming_errors_test.go:167–177` + section-name grep, `sse_integration_test.go:15–17`, `websocket_golden_test.go:102–110`, `du` on jsonrpc/integration_tests, `dsl/_spec` `func Test` count, grpc runtime export grep) — all held. Limitations: single machine, moderate background load on two flagged measurements; Windows CI numbers come from Actions history, not local execution; per-file coverage attributes only in-package execution (cross-package exercise under-counts, caveated where it matters: dsl, codegen/template, codegen/codegentest, codegen/cli).
