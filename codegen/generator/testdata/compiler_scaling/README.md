# Compilation scaling probe

These are reproducible source inputs for the compiler ownership investigation.
The .go.txt suffix keeps this external-module example out of repository package
discovery; the copy commands below give each source its compiling filename. They live
in testdata and do not add a default benchmark or integration gate. Copy the two
Go sources into separate packages in a disposable module; do not run generation
inside the framework checkout.

The design independently varies endpoint count, unique type count, record width
and recursive edges. The probe runs DSL evaluation and generation in one fresh
process, records cumulative allocation and GC/heap counters, then writes an
allocation profile. It intentionally has no timing assertion and forces no GC.

## Reproduction

Use Go 1.27.3 and the framework's pinned dependencies. Set LOOM_REPO to an absolute
checkout path and LOOM_REVISION to the exact pushed commit under examination.
For comparison, create separate disposable modules for each source revision.

```sh
probe_root=$(mktemp -d)
mkdir -p "$probe_root/design" "$probe_root/probe"
cp "$LOOM_REPO/codegen/generator/testdata/compiler_scaling/design.go.txt" "$probe_root/design/design.go"
cp "$LOOM_REPO/codegen/generator/testdata/compiler_scaling/probe.go.txt" "$probe_root/probe/main.go"
cd "$probe_root"
go mod init example.com/scaling
go mod edit -go=1.27.3
go get "github.com/CaliLuke/loom@$LOOM_REVISION"
go mod tidy
go build -o compile-probe ./probe
GOMAXPROCS=10 SCALING_METHODS=16 SCALING_TYPES=1 SCALING_WIDTH=32 SCALING_RECURSIVE=0 \
  /usr/bin/time -l ./compile-probe allocs.pprof
go test ./gen/...
```

For an unpushed local experiment, explicitly replace the framework module with
its absolute source checkout before tidy/build. Record the exact source diff;
a local replacement is not a reproducible remote revision. The `time -l` command
above is for macOS. Its peak RSS and wall/CPU time include profile writing after
the probe's MemStats sample; compare like-for-like runs. Cumulative allocation is
allocated bytes, not simultaneous live memory.

Use a fresh output directory when changing workload shape so old generated files
do not contaminate comparison. Compare every relative artifact path and byte,
including the manifest, across independent processes with the same source identity.
Remote revision pins derive different loom_version metadata in the manifest; enumerate those expected metadata differences when comparing two
remote-pin runs. The recorded exact-byte experiments instead used local source
replacements for both versions, giving both the same `(devel)` version metadata.
To reproduce that comparison, export each exact pushed revision into a separate
source directory and use that directory as its module's local replacement. Record
the pushed revision and confirm the exports have no source edits; this keeps the
source reproducible while equalizing only build identity. Do not discard arbitrary
generated differences as metadata. Compilation
of unchanged output under the same runtime/dependency inputs can reuse previous
compilation evidence. Do not clear shared Go caches or conflate executable build
cost with generator execution. Alternate source order for paired timings, keep
other task builds out of the measurement, and report medians and limitations.

## Workloads

| Case | Methods | Unique types | Width | Recursive | Artifacts |
|---|---:|---:|---:|---|---:|
| Small | 4 | 1 | 8 | No | 16 |
| Reused | 16 | 1 | 8 | No | 20 |
| Unique | 16 | 16 | 8 | No | 20 |
| Wide | 16 | 1 | 32 | No | 20 |
| Recursive | 16 | 1 | 8 | Yes | 21 |

Unset parameters use the source's defaults. Positive method/type/width integers
are required. SCALING_RECURSIVE=1 adds a recursive record edge. These fixtures
sample independent dimensions; they are not a proof of global asymptotic bounds.
The direct field-query, normalization-depth and fingerprint-depth regressions
exercise specific previously multiplicative work with deterministic allocation
checks and recorded semantic equivalence.

## Recorded evidence

See [the compilation architecture](../../../COMPILATION.md) for stage-by-stage
contracts, measurements and caveats. The final comparison uses the shared-method
semantic stage f88a5d32d as its baseline. Final source tree and medians are
recorded there alongside verification and measurement limits. The private real-design
probe is not copied into this repository; these synthetic inputs are public
reproduction evidence for scaling behavior, not a substitute for that workload.

Full CLI measurement is a separate experiment: it includes package discovery,
helper compilation/linking, generator execution and staged installation. Record
whether the Go caches are warm. For warm comparisons, first run each exact
source version once without timing it, then alternate the measured pairs. If
measuring simultaneous process-tree memory,
sample the CLI and its descendants and identify the sampling interval/overhead;
single-process maximum RSS is not a process-tree maximum. The recorded CLI
experiment samples `ps` every 50 ms plus command overhead, so its peak is sampled,
not an exact maximum. CLI/probe command headers differ intentionally; comparisons
between two CLI runs require no normalization.

## Shared error catalog (#1071)

`error_catalog.go.txt` is the issue's standalone HTTP-only reproduction. Use
the disposable module setup above, copying that file to `design/design.go`
and `catalog_probe.go.txt` to `probe/main.go` instead of the original sources.
The catalog probe adds a sampled heap guard and a 45-second time guard. It
writes a `.guard` allocation profile and exits 2 when either guard fires.
Neither the guard nor `GOMEMLIMIT` is a hard RSS limit; sampling can overshoot.

```sh
go build -o compile-probe ./probe
PROBE_METHODS=10 PROBE_ERRORS=52 GOMEMLIMIT=640MiB GOGC=70 GOMAXPROCS=2 \
  /usr/bin/time -l ./compile-probe -max-heap-mib=1024 allocs.pprof
go test ./gen/...
```

Start with a small workload. Increase `PROBE_METHODS` only with the guard in
place; no valid error declarations need to be removed. The guards are probe
instrumentation, not framework runtime policy or CI requirements.

### Qualification evidence

Baseline: pushed commit `2eb9c3e636c72027ec7647430984f4751b4c3644`.
Candidate in this first table: the initial #1071 HTTP capture change, committed
as `8317d47341cf5602a5344e6f2e225efc31bd7307` before the transform extension below.
Both were built from local
source replacements with identical `(devel)` generator identity. The baseline
export contained only the additional artifact qualification test, without
implementation edits. Go 1.27.3, darwin/arm64, 8 GiB host; fresh sequential probe
processes, shared warm build cache, 52 errors, `GOMEMLIMIT=640MiB`, `GOGC=70`,
`GOMAXPROCS=2`, 1 GiB sampled heap guard, 45-second time guard:

| Source | Methods | Result | Generator elapsed | Cumulative allocation | Maximum RSS |
| --- | ---: | --- | ---: | ---: | ---: |
| Candidate | 5 | completed | 2.733 s | 1,291,256,736 B | 368,934,912 B |
| Baseline | 10 | completed | 9.235 s | 3,473,723,512 B | 964,231,168 B |
| Candidate | 10 | completed | 3.288 s | 2,581,155,720 B | 700,727,296 B |
| Baseline | 25 | heap guard | 10.984 s at stop | 4,824,682,056 B at stop | 1,218,871,296 B at stop |
| Candidate | 25 | completed | 13.914 s | 6,509,466,056 B | 1,142,931,456 B |

These are single diagnostic runs, not benchmark estimates. Cumulative allocation
is not peak/live RAM. Generator elapsed excludes profile writing; maximum RSS
includes it. The matched 10-method run used about 27% less peak RSS and 26% fewer
allocated bytes. Candidate allocation grows approximately with method count in
this bounded sample. The stopped baseline is incomplete; its allocation/RSS
must not be compared as a completed 25-method benchmark. GC timing and host load
affect elapsed time and final heap samples. This does not establish a general
memory bound for arbitrary designs.

The actual module-pinned CLI also completed the 25/52 design with the same Go
runtime settings and source replacement:

```sh
go run github.com/CaliLuke/loom/cmd/loom gen example.com/scaling/design -o .
```

That warm-cache diagnostic took 20.732 seconds, with sampled simultaneous CLI
and descendant RSS of 1,150,599,168 B. Sampling used `ps` every 100 ms plus command
overhead; an external 1,700 MiB/100-second guard did not fire. Another bounded
validation build was active, so this is completion/resource evidence, not a
comparative CLI timing result.

`TestLayoutCaptureArtifacts` renders five bounded cases: the catalog; the
existing `ReferenceCompositionDSL` (inheritance, defaults and byte constraints);
`StyledMediaContractsDSL` (codec branches); `UntaggedNullFieldsDSL` (presence and
nulls); and `MultipleViewsDSL` (selected layouts). All 123 generated Go, protobuf
and OpenAPI files were byte-identical between baseline and candidate, and across
two independent candidate processes. There are no intended generated differences.
Those exact candidate packages compiled with `go test ./...`. Protobuf tools
matched the repository pins: protoc 36.2, protoc-gen-go 1.36.12 and
protoc-gen-go-grpc 1.6.2.

To repeat the bounded comparison, set `LOOM_LAYOUT_CAPTURE_ARTIFACTS` to a fresh
output directory and `LOOM_DIR` to the exact source directory, then run:

```sh
go test ./codegen/generator -run '^TestLayoutCaptureArtifacts$' -count=1
```

Compare each case's `gen/` tree byte for byte; `go.mod` replacements intentionally
name different source directories. Compile the generated modules themselves.
Normal direct regressions cover shared registry identity, allocation and key size,
false/missing layout facts, sibling overrides, fresh unprepared stream layouts,
media plan isolation, invalid selections and runtime/schema example ownership.
Both checked-in HTTP and JSON-RPC ticktock fixtures also regenerated to their
existing exact trees and built in local-only temporary copies. The bounded
TLA models and their limitations are linked from `codegen/COMPILATION.md`.

### Transform field analysis extension

The same issue/PR also removes discarded descendant copies from transform
matching. The old path called `NewMappedAttributeExpr` (a deep mutable copy),
then replaced its copied children with the original occurrences. Every field
planning pass also repeatedly recomputed mapped requiredness. The new
private `transformFields` view captures `expr.ObjectFields` once per walk and
borrows the winning original children from the stable transform-owned graph.
The existing mutable mapping API remains usable by default-plan callers.

`TestTransformFieldQueriesDoNotCopyDescendants` failed before implementation:
matching an object against itself allocated 285 objects with an unrelated child
of width 16, and 3,293 at width 512. It now allocates 55 at either width. Direct
controls cover mapped order, repeated exact keys, selected child identity,
inherited defaults, named occurrence versus declaration layout, source
immutability and rejected nonobjects. The existing transform suite covers
nullable/presence, admitted defaults, recursive helpers and rejected conversions.
Review added a failing winning-exact-key default regression (replace, supply and
remove a default). Defaults now resolve directly on the selected physical child
through the shared `AttributeExpr.EffectiveDefault` resolver, rather than looking
up its original key again and accidentally finding the first duplicate.
The field-facts model's EagerCopied negative control reproduces the discarded
construction work; Owned passes. Its assumptions and limitations are beside it.

Fresh sequential probe processes compared the initial PR commit above with the
transform extension on the same Go 1.27.3 darwin/arm64 host and runtime settings
as the first table. Ten-method runs used the probe's default 768 MiB sampled
heap guard; 25-method runs explicitly used `-max-heap-mib=1024`. All completed:

| Source | Methods | Generator elapsed | Cumulative allocation | Maximum RSS |
| --- | ---: | ---: | ---: | ---: |
| Initial PR | 10 | 3.408 s | 2,581,466,144 B | 682,508,288 B |
| Transform extension | 10 | 4.201 s | 2,199,508,616 B | 638,418,944 B |
| Initial PR | 25 | 15.377 s | 6,509,975,928 B | 1,082,130,432 B |
| Transform extension | 25 | 17.029 s | 5,556,232,688 B | 1,130,577,920 B |

These remain single diagnostic runs. Allocation fell by about 15% at both sizes,
but large-case peak RSS did not improve. This removes transient copying and GC
work; it does not eliminate the live graph/constraint footprint or establish a
fix for a consumer's reported 12+ GiB workload. The remaining allocation profile
is dominated by occurrence/constraint capture, not transform mappings.
Other bounded validation builds were active, so these timings are not comparative
performance estimates. The final source was measured again after the review fix;
the 25-method final run's maximum RSS was about 4.5% higher than the initial PR.

All 123 artifacts were compared again against both the original main baseline
and the initial PR, plus two process-isolated final generations: exact bytes,
no intended differences. The exact final modules compiled, including HTTP and
gRPC; the generated 25/52 catalog compiled too. Both maintained SSE fixture
trees regenerated unchanged and built. Affected codegen, representation,
transport IR and OpenAPI package tests, focused HTTP checks, and generated
selected-body and typed-collection default tests passed. Existing selected
Vacuum positive/negative evidence remains valid because the compared specs are
unchanged; this extension changes neither schema contracts nor encoding policy.

### OpenAPI acquisition journal extension

A consuming-application timeout report on commit `452b454af76ac3dc0545e5713401e0395ace847d`
required separate platform and generator investigation. That exact framework
revision completed on an exclusive runner outside the earlier shared soft-limit
throttling. This does not retroactively diagnose every wait in the interrupted
runs, nor establish that this application fits their shared memory budget.

A fresh CPU/heap/stack probe of the complete consuming design located repeated
OpenAPI acquisition checkpoints. `acquisitionRollback` cloned all accumulated
registration indexes, reference buckets and annotation owner maps per outer
schema scope. About 29% of sampled allocated bytes came from those snapshots.
The new journal records changed entries only, restores them in reverse order on
error/panic and relinquishes them on success. Mutable reference buckets detach
before element replacement and scoped append, including spare capacity that
could retain rolled-back bindings; lazy indexes restore nil on failure. Schema
completion, canonical comparison and example callbacks retain their contract.
The [journal model](../../../../http/codegen/openapi/internal/ir/tla/acquisition_journal/README.md)
records the design assumptions, positive/negative checks and implementation seams.

The direct checkpoint regression failed at 57 versus 4,133 allocations for 16
versus 2,048 prior annotation owners, without any new registrations. It now uses
one allocation at either size. Scope outcome tests cover nested traversal,
repeated write/delete, present-nil entries, new entries, reference promotion with
spare slice capacity, annotation maps, lazy allocation, errors and panics.

One sequential consuming-design qualification per source used Go 1.27.3 on
linux/amd64, `GOMAXPROCS=2`, `GOMEMLIMIT=8GiB`, `GOGC=10`, normal concurrent GC,
an owned hard budget, a 7.5 GiB sampled process-group RSS guard and a 20-minute
deadline beginning only after exclusive capacity acquisition. All ancestor limits
were checked; shared limits and source checkouts were not changed. Source and
output copies, workspace selection and profiling were private qualification
inputs; application package versions and lockfiles were preserved.

| Measurement | Initial PR `452b454af` | Journal extension |
| --- | ---: | ---: |
| Instrumented worker wall time, including profile finalization | 148.776 s | 115.243 s |
| Initial file analysis | 117.693 s | 83.944 s |
| `generator.Generate` elapsed | 140.045 s | 106.635 s |
| Total allocated bytes at generator completion | 43,158,192,304 B | 30,721,330,840 B |
| Instrumented worker sampled peak RSS | 6,157,828,096 B | 5,730,652,160 B |
| Ordinary `go generate ./design` elapsed, including CLI build | 209.131 s | 127.584 s |
| Ordinary CLI sampled peak process-group RSS | 6,087,847,936 B | 5,807,468,544 B |

These are single diagnostics, not benchmark estimates. Helper-build cache and
disk costs varied; full CLI timings do not isolate generator improvements.
Profile-writing overhead belongs only to the instrumented runs. Allocated bytes
fell by about 29%. The ordinary CLI's peak RAM was only modestly lower (about 5%),
remaining around 5.4 GiB.
Both ordinary runs had zero memory-high and OOM events in their owned scope.
Substantial live occurrence/constraint graphs remain; neither these results nor
the journal establish a universal memory bound or fitness for a tighter shared
soft limit. An increased timeout alone is not the implemented correction.

Both ordinary runs generated all 335 consuming-application backend artifacts;
the journal version matches the prior PR byte for byte. Frontend regeneration,
the API contract hash check and strict API audit passed. The baseline audit first
identified an existing derived evidence-count mismatch; its mechanical evidence
was regenerated only in the private validation copy, without changing route
identities or adoption dispositions. Six targeted generated packages compiled
with `GOWORK=off`, `-mod=readonly` and the application's original dependency pins.
All eight original application dependency/lock files were checked separately.
Private source, raw profiles and operational paths are not public fixtures.

The bounded artifact matrix now also includes the existing
`DiscriminatorHintDSL`, exercising tagged/untagged mapping, form and multipart
representations. All 146 generated Go/protobuf/OpenAPI files match original main,
the initial PR and two process-isolated journal generations exactly. Those exact
generated modules compile; both maintained SSE fixtures regenerate unchanged and
build. Full affected OpenAPI IR/v3 package tests, acquisition failure/claim
controls, selected pinned Vacuum positive/negative checks and scoped linters pass.
There are no intended generated-contract differences.
