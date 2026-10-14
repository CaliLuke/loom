# Race execution ownership (#941)

## Evidence and accepted scope

The race job on b4e659f1f82789636e8d53489f25920591226848 exhausted the
40-minute `http/codegen` package timeout while a newly started test was actively
formatting generated source. The same runner spent 2128.614s on
`codegen/generator` and 2222.492s on `grpc/codegen`. This establishes a failing
aggregate execution budget, not a deadlock or data race in the active test.

The shared race invocation currently schedules these independent generators and
their nested Go builds on one runner. Separate the three whole packages and the
remaining root-module packages into four CI lanes. Preserve every test, complete
within-package shuffle, race instrumentation, count=1 and the 40-minute timeout.
Local `make test-race` continues to select the entire suite by default. Separation
reduces competing nested builds; no specific speedup is assumed or promised.

## Sequence

- [x] Preserve the failed job and bounded source/population evidence; file #941.
- [x] Add one shared lane selector/executor and make the existing Make target
  delegate to it. Reject unknown/empty lanes and incomplete package inventories.
- [x] Set the required race workflow to four lanes with fail-fast disabled; keep
  release eligibility dependent on the complete race matrix.
- [x] Prove disjoint/exhaustive package coverage using the actual `go list -race ./...`
  inventory, preserving exact test flags and default invocation. Test unknown,
  empty and missing-owner admission without running the full suites locally.
- [x] Run bounded native lane smoke checks and the existing CI contract lint.
- [x] Independent exact-diff review and atomic delivery as `a23bc990f`. All four
  remote race lanes passed; #941 is closed.

## Acceptance and limits

The selector is a deterministic package partition, not a test-name partition.
Each selected package is run in full; no new test skips or transport harnesses
are introduced. The default target retains `./...`. The CI lanes must cover each
current root-module package exactly once, and new ordinary packages enter the
remaining lane automatically. A missing dedicated package is an error rather
than silently reducing coverage. The finite timeout still detects a stuck
package; remote lane duration is the final capacity evidence. Existing source
selection and subprocess cleanup contracts remain unchanged.

## Validation receipt

Against code-identical parent 506527c76 (the later 69efebc93 changes only
AGENTS.md), the shared contract check partitions all 93 root-module packages
exactly once. It checks the unchanged default command, all four complete lane
selections, exact race/shuffle/count/timeout flags, Make forwarding, workflow
lane binding, missing dedicated packages, empty remainder, invalid lane and
preserved nonzero test status. The existing CI contract lint passes, including
its hook and integration-runner assertions. Shell syntax checks pass.

Bounded representative race/shuffle tests pass in HTTP codegen (2.241s), gRPC
codegen (1.630s), generator (1.574s) and DSL as the remaining-lane representative
(1.476s). These smoke checks establish the selected invocation remains usable;
they do not establish full-lane capacity. No framework generator output changes,
fixtures or production Go behavior are part of this correction. Full remote
matrix completion remains the delivery evidence for capacity.

Independent review caught a build-mode mismatch: discovery must also enable
`-race`, or a package containing only race-tagged files is omitted. The owner
and inventory contract now use that mode. A minimal real Go module containing
an ordinary root, the three dedicated packages and a race-only package proves
that the remainder includes the race-only package; test execution is stubbed
but discovery uses real Go. Baseline Make assertions explicitly set default
options so caller-selected lanes/timeouts do not invalidate contract lint.

Remote capacity acceptance at `a23bc990f01c2c0c57d33df5ba15d2bff99c3400`
passed in [run 38614785410](https://github.com/CaliLuke/loom/actions/runs/38614785410):
gRPC 11m09s, generator 11m44s, HTTP 13m48s and remaining packages 17m16s.
Every lane executed its complete package set under the unchanged budget.
The immutable snapshot and concise receipt are
`/tmp/loom-941-four-race-lanes-passed.json` and
`/tmp/loom-941-capacity-receipt.json`. This proves the partition's remote capacity;
final required CI on the later `a0642853c` source remains separately pending.

Exact source inputs, independent review, reproduction and capacity evidence are
archived at `/tmp/loom-941-durable-inputs.tar.gz`, SHA-256
`d2cf68fd664e491e73c855cb14424d82186cf7717f40ec4f0f7246cf12a9488e`.
Activity-checked cleanup removed 11 inactive temporary paths; receipt:
`/tmp/loom-941-cleanup-receipt.json`.
