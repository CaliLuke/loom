# Compilation ownership and resource use

## Objective

Compilation should retain semantic information once, derive explicit target
representations, and release work when its consumers finish. Correctness includes
authored defaults, constraints, examples, error ownership, custom codecs, plugin
ordering, deterministic output and atomic installation by the CLI.

This is the architecture contract for the compiler redesign. It distinguishes
the intended design from the implementation checkpoints below. A compiler is
not improved merely by adding an arena, a cache or more workers: each mechanism
must remove measured work or bound a measured resource without weakening these
contracts.

## Current boundary

`generator.Generate` loads roots, fingerprints the design, runs preparation
plugins, analyzes service and transport output, runs generated-file plugins,
admits and merges paths, renders files and computes the output manifest. The CLI
owns staged installation. `generationFiles` shares one service `SemanticData`
per root; backend names and synthesis domains remain separate.

`ValueContext.NewOccurrence` snapshots a mutable attribute graph independently.
Its declaration table is local to that capture. `NewValuePlan` captures another
target graph, even when another plan just captured the same finalized target.
HTTP schema, service-anchor and target-example plans therefore repeat capture
and constraint analysis. Generated sections retain backend data until rendering;
the renderer queues every file and uses the host CPU count for concurrency.

The existing generated-file plugin API can inspect and replace the entire file
collection. That is an explicit whole-compilation barrier. Its arbitrary Go
callbacks cannot be assumed pure or replayed safely from a persistent cache.

## Design principles and sources

- Share immutable compiler data within an explicit owner. Rust's compiler uses
  interning and compilation lifetimes to avoid repeated type allocation. Loom
  should use ordinary Go-owned storage with explicit lifetimes first; an unsafe
  allocator is not required by this principle.
  [Rust compiler memory management](https://rustc-dev-guide.rust-lang.org/memory.html)
- Reuse analysis only over immutable inputs, with every semantic dependency in
  its key. Rust's query model requires immutable keys/results and pure providers.
  Custom example generators are effects, so their draws are outside reusable
  semantic analysis.
  [Rust query evaluation](https://rustc-dev-guide.rust-lang.org/queries/query-evaluation-model-in-detail.html)
- Separate transformations from their analyses and make invalidation explicit.
  LLVM's pass manager provides this distinction. Loom's capture boundary must
  precede reusable analysis; later target changes require a new capture.
  [LLVM pass management](https://llvm.org/docs/NewPassManager.html)
- Keep representations appropriate to their abstraction level. Service semantics,
  transport layout and rendered syntax have different owners. MLIR provides a
  useful precedent for explicit lowering between representations; adopting MLIR
  itself is not necessary for a Go source generator.
  [MLIR architecture paper](https://arxiv.org/abs/2002.11054)
- Separate dependency scheduling from decisions about reuse. Persistent reuse
  needs complete input dependencies, not just an output filename or design hash.
  [Build Systems a la Carte](https://www.microsoft.com/en-us/research/wp-content/uploads/2018/03/build-systems-5ab0f42d0f937.pdf)

## Semantic and representation ownership

The evaluated design remains the authored source of truth. A compilation owns
captures whose builtin structure and ancestry are immutable; a backend owns
naming, layout and sampling effects. Opaque/custom host values retain the existing
borrowing contract: capture does not invoke codecs or make arbitrary Go values
immutable. Materialization remains a separate effect and is not deduplicated by
target sharing. A target
capture owns its finalized declaration shape and effective constraints. A plan
combines a source occurrence, a target capture, a selection and explicit codec,
presence, visibility and unknown-member policies.

Declaration identity and occurrence identity are different. Reusing a declaration
must not conflate two method roles, two selected paths, defaults, local examples,
constraint provenance or sampling streams. Recursive declarations remain finite
graphs. Equal spelling, Go names or schema shape do not establish semantic identity.

The planned storage boundary is a compilation-owned immutable declaration store
plus occurrence/projection records. Its sharing domain must be frozen explicitly.
`NewOccurrence` over a mutable caller attribute must continue to observe the state
at that call; a pointer-keyed cache that silently returns an older capture is
invalid. Failed construction must not publish a partial declaration or poison
later captures. Independent compilations must not share retained design graphs.

### Captured target contract

The first representation boundary permits `ValuePlanRequest` to identify either
a mutable `Target` to capture now or an immutable `TargetOccurrence` to reuse.
These are distinct operations, not implicit cache modes.

| Input | Behavior |
|---|---|
| Only `Target` | Capture its current finalized state, then build the plan. |
| Only valid `TargetOccurrence` in the source context | Reuse that snapshot; do not capture or validate its declaration again. |
| Both target forms | Reject the request before building a plan. |
| Neither target form, invalid handle or foreign context | Reject with a target/context diagnostic. |
| Captured target unrelated to the selected source | Reject by captured ancestry, even when shape or names match. |
| Caller mutates original target after capture | Existing capture and all plans made from it retain the captured shape, constraints and ancestry. A fresh mutable-target request sees the new state. |

Runtime field/container/codec policies remain plan inputs. They identify exact
authored locations; their pointer fields are identity tokens, not permission to
read mutable attributes. Plans over the same capture can differ by policy and
must remain independent. Copy policy collections and mutable policy values into
owned plan state where they are retained.

With unchanged inputs, raw-target and captured-target planning must produce
equivalent schemas, examples, defaults, validation and diagnostic meaning, and
byte-identical generated artifacts. The intentional distinction is subsequent
mutation: only a fresh capture observes it. This includes source and descendant
ancestry changes, not merely target root shape. Ancestry is normalized into owned
identity data during capture; planning cannot traverse live provenance chains.
Neither route changes custom materialization calls or example sampling streams.

HTTP representation analysis captures its final target after layout and view
selection, then shares that capture between schema, anchor and example plans.
Earlier transport-local example captures cannot substitute for this later
layout capture. A rebuilt layout creates a new target capture and invalidates
all plans/annotations keyed to the previous target. Standalone schema analysis
can use its existing capture as both source and target when their shapes agree.

## Artifact planning and rendering

Planning establishes paths, write intent, removals, warnings and same-path
composition before output installation. Rendering consumes an admitted plan.
Scheduling must preserve existing section/declaration order and same-path merge
semantics. Parallel completion order must not choose public names or diagnostics.

The scheduler owns a bounded set of active render jobs. Completed artifacts must
release render-only state when no remaining consumer needs it. Immutable shared
state remains live only for its remaining consumers. A bounded worker count alone
does not establish a total memory bound: analysis, queued closures, the largest
artifact and arbitrary extension state must be included in any stated bound.

Current plugins retain their documented preparation and file-rewrite behavior.
Their whole-file barrier must be reported honestly in resource measurements.
A planned-artifact extension contract must define declared dependencies, ordering,
ownership and mutation scope before claiming streaming or incremental behavior.
Declaration-store admission (including atomic recursive-group publication),
deterministic error selection together with cancellation, and the extension API
need stage-specific contract refinement and review before implementation. They
are architectural direction here, not completed designs. No hidden adapter may
claim purity for an opaque callback. Changes to plugin
contracts require an inventory of actual integrations and a concrete migration.

## Reuse, effects and failure

In-process reuse belongs to the capture/analysis owner. Persistent reuse is
admitted only for inputs with complete dependencies, including framework revision,
toolchain, design, generator options and plugin dependencies/effects. Existing
arbitrary DSL and plugin Go code cannot be treated as hermetic. A persistent
cache is not a prerequisite for safe, efficient one-shot compilation.

The CLI's staged-installation guarantee, skip-existing intent and removal rules
remain unchanged. Invalid artifact paths and policy conflicts fail before
rendering effects. Render failures retain contextual diagnostics. Direct
`generator.Generate` calls can leave partial output, including a rendered
manifest; they do not provide a transaction. The CLI stages that output and must
not install any of it on failure. The eventual cancellation contract must stop
queued work and join owned workers before returning. Arbitrary plugin side
effects remain plugin-owned.

## Evidence and completion criteria

Measure the generator process separately from CLI package loading, Go compilation
and staged installation. Report cold and warm toolchain work separately; do not
clear shared build caches to manufacture a cold result. Retain source revision,
toolchain, workload parameters, output digests and summarized results. Run paired
fresh processes, alternate baseline/candidate order, and keep concurrent test
builds out of the timing experiment.

Track total allocation, live heap at phase boundaries, peak process RSS, CPU and
elapsed time. A diagnostic forced-GC live-heap sample is a separate experiment
and must not be mixed with normal latency/RSS results. Scaling fixtures vary
unique declarations, reuse count, recursion, endpoints and artifact sizes
independently. Count capture/analysis work so accidental multiplicative growth
can fail a deterministic regression without a wall-clock threshold.

The redesign is complete when immutable representation reuse and ownership are
implemented, output work has explicit admission/lifetimes and bounded scheduling,
plugin limitations are accurately supported/documented, repeated work scales
with unique declarations and actual occurrences rather than redundant passes,
and paired real-project/scaling evidence demonstrates an improvement without
unexplained semantic or performance regressions. No claim of universal optimality
or constant-space compilation follows from these criteria.

Proof obligations include independent occurrence identity despite shared storage;
immutable capture under caller mutation; recursive termination; no cross-context
admission; independent codec/presence/sampling policies; deterministic diagnostics
and bytes; no release before last use; and no CLI artifact installation on failure.
Bounded models establish only their stated ownership/scheduling assumptions;
direct and generated-code tests establish implementation behavior.

## Recorded checkpoints

- `d2b2d1f02`: indexed value-plan membership removed repeated reachability walks.
  On the initial real-project probe, allocation fell from about 12.16 GB to
  6.57 GB, without a demonstrated peak-RSS reduction.
- `f88a5d32d`: shared method semantic ownership, with independent backend naming
  and custom sampling. Three quiet paired runs: allocation 6,568,510,568 to
  6,544,618,016 bytes; median RSS 794,722,304 to 760,987,648 bytes; elapsed
  4.31 to 4.28 seconds. Earlier contended runs showed flat RSS. Generated bytes
  were identical; this is a modest workload-sensitive gain. These figures
  exclude compilation of the generator executable.
- Captured-target candidate against `f88a5d32d`: three alternating fresh-process
  pairs on the same 63-artifact design reduced median allocation from
  6,544,764,256 to 6,180,359,512 bytes (5.6%) and peak RSS from 770,949,120 to
  637,485,056 bytes (17.3%). CPU medians were 9.26 and 8.84 seconds. Wall medians
  were 4.90 and 5.28 seconds, with competing machine workloads and a baseline
  outlier of 11.66 seconds; elapsed-time improvement is not established. All
  six output trees matched the baseline exactly. Width benchmarks at 8, 64 and
  256 fields reduced per-plan allocation by 58–62% when reusing a capture.
