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
Its declaration table is local to that capture. `NewValuePlan` accepts either a
mutable target to capture or an existing immutable target occurrence. HTTP schema,
service-anchor and target-example plans share the finalized target capture. Generated sections retain backend data until rendering;
the renderer admits a private artifact plan and bounds active work independently
of the total artifact count.

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

## Immutable compiler queries

The captured-target stage removes repeated construction. Consumers still ask
read-only questions by copying complete declaration graphs or searching every
semantic descendant for each target member. The current real-project profile
attributes about 603 MB to repeated association searches and 169 MB to
`ValuePlanNode.Attribute` copies (sampled allocations, overlapping callers).

### Contract

`ValuePlan` owns the exact semantic-to-representation association relation built
by `valuePlanBuilder.associate`. Queries must derive from that relation. HTTP
example preparation must not reconstruct it from names, shapes or ancestry.

Add `ValuePlan.OccurrenceForTarget(root, target)` to return the unique associated
semantic occurrence in the region reachable from `root`, including `root`.
Reachability follows captured alias, object member, union alternative, array
item and map key/value edges. Cycles are finite graph reachability; repeated
paths to the same occurrence count once. Constraint inheritance and stored result views are not
structural edges merely because their nodes appear in the capture node table. Named aliases remain distinct semantic owners.

Admission and outcome rules, in order:

1. Reject an invalid plan, an absent root, or a root from another context/capture.
2. Reject an absent target or a target outside this plan's reachable target
   subgraph. A derived subplan rejects unreachable nodes; recursive paths may
   make apparent ancestors or siblings reachable and therefore admissible.
3. Consider only exact associations to the admitted target whose source is
   structurally reachable from root.
4. No association returns `ErrValuePlanAssociationNotFound` (wrappable).
5. Multiple distinct source occurrences, or multiple associations for one source
   and target, return an ambiguity error. An exact current plan association has
   the same precedence as `ForOccurrence` for that source/target pair only; it
   cannot suppress another reachable owner. A selected-body target queried from
   the original source root remains ambiguous when both the root-with-selection
   and the selected member associate with that target.
6. A unique admitted source returns a handle in the original context/capture.
   A subsequent `ForOccurrence` must recover the corresponding projection.

No query resolves examples, invokes codecs, synthesizes values, changes selected
union branches, or publishes a partially built plan. Raw mutable attributes
never participate in query ownership.

The plan stores the reverse association index once, shared by its subplans.
The captured semantic graph owns a lazily initialized reverse structural-edge
index. The index is derived only from immutable captured edges and published
once with synchronization; concurrent reads return the same semantic answers.
Traversal walks parents of candidate owners until it reaches the requested
root. It does not walk unrelated descendant subtrees or enumerate all semantic
nodes for every target. No transitive-closure table or cross-compilation cache
is retained. Space is linear in captured edges and associations. Worst-case
reachability can still visit the whole graph, especially dense recursive graphs;
no constant-time reachability claim is made.

`ValuePlanNode.MetaLast(key)` returns the last captured metadata string and its
presence, with the same lookup behavior as `MetaExpr.Last`; strings are immutable.
`ValuePlanNode.IsInlineHTTPBody()` answers the existing wrapper predicate from
captured attributes without exporting a mutable pointer. Invalid handles return
absent/false. `Attribute()` retains its owned-copy contract for callers that
actually need a mutable graph. No borrowed `AttributeExpr` is exposed.

HTTP member/branch searches use the shared query. The ordered alias-chain
lookup in `associatedUnderlyingProjection` keeps its existing first-match
behavior, excluding its starting root; it is not replaced by this exhaustive
unique-owner query.

HTTP example preparation derives child suppression from `MetaLast`; root
suppression continues to use the explicit finalized attribute passed by its
caller. Example selection, traversal order, scope strings and sampler/custom
materializer calls remain identical. OpenAPI wrapper alignment uses the captured
predicate. Byte-schema projection can keep `Attribute()` where it requires the
full declaration; replacing that remaining path is deferred until it has an
appropriate immutable schema query contract.

### Equivalence and proof obligations

All successful HTTP, JSON-RPC and OpenAPI bytes must remain exact, including
hidden fields, selected bodies, examples, defaults, custom codecs, aliases,
recursive declarations and independent documentation targets. Existing invalid
association cases remain rejected with contextual diagnostics; the HTTP layer
wraps shared query errors with the member/branch position.

A bounded model must compare indexed reverse reachability to exhaustive forward
reachability, including recursion, disconnected owners, multiple owners and
same-pair ambiguity. Unsafe omission of scope filtering must produce a
counterexample. Direct tests must establish foreign/context rejection, subplan
restriction, selection association precedence, repeated-path deduplication,
cycle termination, frozen metadata under caller mutation, and concurrent query
publication. Width scaling must show that successful sibling queries do not
allocate a whole-graph search per member. Generated output, allocation, peak RSS
and CPU/elapsed time must be measured against the preceding stage.

## Semantic memo lifetimes

Memo tables belong to their immutable key owners. Resolution tables belong to
supplied source snapshots; synthesis tables belong to occurrence capture graphs;
projection tables belong to constructed plan graphs. The context supplies
synchronization without rooting every temporary source, capture and target plan.
A new plan and its copies/subplans share one memo owner created before publication;
independently constructed plans have independent owners. Entry tables initialize
lazily. Keys, admission, outcomes and callback counts remain unchanged.

Projection publishes a pending slot before invoking custom materialization outside
the lock. Readers wait for completion. Success and ordinary failure publish the
retained result; a panic propagates to its creator while releasing readers with
the existing incomplete-materialization failure. This is the existing effect
contract, not permission to replay callbacks. Reentrant same-key projection
remains unsupported. Resolution and synthesis retain their existing context
serialization and role/sampling-scope isolation.

Live handles and active calls retain the owners they need. Once they disappear,
Go's tracing collector can collect owner/cache cycles. No weak cache, eviction,
finalizer, forced collection or terminal phase-close API is used in production.
A live source can retain sibling resolutions, a capture retains scoped synthesis,
and a subplan can retain its full association graph. The context still retains
authored-example source associations for cross-capture identity, which can retain
those sources' resolutions. These are explicit limits, not a total-memory bound.

The [memo lifetime model](../expr/tla/memo_lifetime/README.md) compares global
retention, owner-local reachability and unsafe early release. Concrete weak-pointer
GC tests keep the context alive while dropping owners; identity, scope, concurrency,
codec-panic and subplan tests check behavior beyond the model. Weak pointers are
observations in tests only.

## Normalization input ownership

HTTP transport normalization captures its detached input once before changing
that copy. Lowering carries the corresponding immutable occurrence through named
declarations, object members, map keys/values and array elements. Effective
constraints come from this input snapshot; the mutable output cannot redefine
its own semantic input. Shared declarations keep the existing type-ID recursion
rule while each edge keeps local defaults, validation and nullability.

There is no cache across calls. A later call observes later caller mutations;
failed capture publishes no normalized output. Missing captured positions or
mismatched object members are internal errors. Canonically named types, result
types, first-class unions, inherited examples and physical alias flattening keep
their existing rules. Clone behavior and the finalized-input precondition remain.

The [normalization pass model](../http/codegen/internal/transportir/tla/normalization/README.md)
compares one capture with repeated subtree capture. The depth allocation
regression and shared/recursive declaration controls check the Go implementation.
This removes redundant input analysis; it does not bound arbitrary constraint
solving or all other compiler passes.

On the measured real-project design, normalization capture calls fell from 19,080
recursive captures to 1,172 pass-input captures; captured node visits fell from
156,230 to 45,492. These are diagnostic counts, separate from latency/RSS runs.
At nested depths 16/64/256 the pass allocates 382/1,406/5,452 objects; the previous
pass allocated 2,216/27,912 at depths 16/64. The regression checks growth rather
than absolute timing or a machine-specific allocation ceiling.

Three alternating fresh-process pairs against the bounded-renderer stage on the
same 63-artifact project and Go 1.27.3 reduced median allocation from
3,763,719,512 to 3,506,363,120 bytes (6.8%), RSS from 666,959,872 to 646,627,328
bytes (3.0%), elapsed time from 2.86 to 2.46 seconds (14.0%) and CPU from 5.24
to 4.84 seconds. No task builds ran during this window; other machine activity
remained. All output trees matched exactly. The same five representative designs
also matched twice across processes and their generated code compiled.

## Checked fingerprint inputs

Canonical OpenAPI fingerprinting carries a captured semantic position alongside
its encoding attribute. Each finalized input receives one capture and mandatory
whole-input documentation-plan admission, including when it has no annotations.
This preserves rejection of duplicate emitted JSON names and other representation
conflicts. Structural visits reuse captured children. Actual defaults and enum
clauses still receive independent projection plans and fresh source identities;
sharing analysis never authorizes replaying or suppressing custom materializers.

A copied alias with only a title change keeps the original captured policy-origin
attribute. A changed validation/nullability overlay and a synthetic bytes-base
wrapper are new semantic inputs with fresh capture/admission. Do not attach
policies built from a copied parent pointer to its original capture. This pass
does not use `ForOccurrence` to bypass ambiguous shared-declaration associations.
Raw fingerprints used for seeds and union sorting retain their existing encoding.
Valid checked fingerprints retain their version, bytes and component names.

Nil roots retain the existing nil fingerprint without effects. A capture, plan or
annotation error retains the first analyzer error and stops further checked
annotation effects. Structural encoding still returns a full-length error-path
hash for existing callers; the analyzer must reject the resulting document.
The existing finalized-shape precondition remains for malformed nil members and
broken named declarations. Invalid-input failure ordering can become earlier.
There is no cross-call mutable-pointer cache.

The [checked fingerprint model](../http/codegen/openapi/internal/ir/tla/fingerprint_input/README.md)
compares retained input with redundant analysis, stale derived semantics, copied
policy origins and skipped representation admission. Recorded fingerprint and
codec-count tests check concrete equivalence beyond those abstractions. Ordinary
structural traversal needs one capture/admission; actual annotation shapes and
new semantic overlays still incur their own planning work.

An initial three-pair comparison reduced allocations but showed a wall-time
regression with nearly flat CPU. Inspection found extra per-object bookkeeping
in the raw encoder; its depth benchmark now retains the original allocation
counts. A five-pair follow-up on the same design, with no task builds, allocated
3,506,527,688 versus 3,364,539,368 bytes (4.0% less). Median RSS was 640,401,408
versus 628,588,544 bytes (1.8% less), elapsed 2.91 versus 2.52 seconds (13.4%
less), and CPU 5.18 versus 4.85 seconds (6.4% less). Every output tree matched.
The earlier wall regression did not persist; these measurements do not isolate
how much the raw-path correction versus machine variation contributed. The
five representative designs also matched across processes and compiled.

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
Declaration-store admission (including atomic recursive-group publication) and
a phased extension API remain architectural directions requiring their own
contract refinement, integration evidence and review. No hidden adapter may
claim purity for an opaque callback. Changes to plugin
contracts require an inventory of actual integrations and a concrete migration.

The admitted plan rejects nonlocal destinations, case-folded collisions,
file/descendant conflicts and reserved manifest paths before render effects.
Exact same-path contributions still compose in their original order. Its private
pending slice relinquishes each file on dispatch; completed jobs return only
metadata. Failure drops unstarted references and joins active callbacks. It never
clears sections owned by callers or plugins. The default is min(4, GOMAXPROCS),
with a positive per-invocation `WithRenderWorkers` override. This is an active-file
bound, not a byte bound. Deterministic outcomes across worker counts assume stable
filesystem state and independent per-file effects; arbitrary callbacks cannot be
interrupted. See the [scheduler model](generator/tla/render_scheduler/README.md).

## Scalar field analysis

Read-only field queries consume expr.ObjectFields, a pass-owned snapshot of
immediate object fields. It records required/default presence, native primitive
pointer eligibility, transport names and the winning original field index/key.
It retains no attributes or raw defaults and never copies unrelated descendants.
Requiredness resolves once per object; individual field lookup does not allocate.
The existing AllRequired resolver still has its own normalization cost.

Exact keys collapse at their first position with their last attribute before
logical names collapse by that same rule. Fields retain mapped behavior, including
qualified names and dangling required names. An iterator exposes scalar selections
in that order. The consuming pass owns a stable input graph and reads its selected
child; subsequent input mutation requires new facts. Actual mutable mappings keep
fresh queries and explicit Map overrides. No pointer-keyed cache is introduced.

AttributeContext.FieldPresence consumes the narrow FieldFactsProvider query.
Existing mapped callers remain valid; builtin nil providers preserve absent-parent
behavior. Validation and HTTP layout share frozen facts instead of asking a deep
mutable copy to answer scalar questions. Raw ServiceFieldLayout and captured
ObjectFieldFacts.ServiceLayout share one native storage policy, preserving exact
raw keys versus mapped interpretation and lazy primitive-pointer queries.

The [field facts model](../expr/tla/field_facts/README.md) contrasts immutable
snapshots with borrowed mutable facts and repeated descendant copying. Direct
parity, selection-order, mutation, nil, recursive-release and allocation checks
cover the concrete semantics. Eight required-field checks previously allocated
1,177 versus 13,201 objects when an unrelated descendant widened from 16 to 512
fields; the snapshot path allocates 59 in both cases.

Five alternating fresh-process pairs against the checked-fingerprint stage on
the same 63-artifact design reduced median allocated bytes from 3,364,621,472
to 3,065,041,760 (8.9%) and peak RSS from 660,848,640 to 626,802,688 bytes
(5.2%). Median elapsed time was 2.61 versus 2.47 seconds and
CPU 4.87 versus 4.40 seconds. No task builds ran in this window;
other machine activity remained. Small elapsed differences do not establish a
general speed improvement. All real-project output trees matched exactly; the same
five representative designs matched twice across processes and compiled.

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
not install any of it on failure. The scheduler stops new dispatch after the first observed failure and joins
started workers before returning the earliest failing plan index. Arbitrary plugin side
effects remain plugin-owned.

## Exact constant representation

The numeric owner stores ordinary finite constants inline: sign, a uint64
coefficient and signed 32-bit powers of two and five. Larger coefficients or
exponents use private immutable arbitrary-precision storage. Copies may share
that storage; inputs and returned values cannot mutate it. This follows the
small-value/large-value representation principle used by
[LLVM APInt](https://llvm.org/doxygen/APInt_8h_source.html), adapted to Loom's
exact finite numeric contract.

Native integer conversion and finite IEEE binary floating conversion normalize
without heap allocation. Comparison reconstructs a native unsigned magnitude
only when checked shifts and multiplication prove that it fits. Other cases use
the existing exact arbitrary-exponent comparator. No overflow, rounding, altered
JSON/DSL grammar or changed carrier admission is permitted. Canonical keys and
text are independent of storage, including signed zero and forced promotion.
There is no global interning cache.

The [bounded arithmetic model](../internal/exactnum/tla/compact/README.md)
checks normalization and checked reconstruction, with a wraparound negative
control. Independent big.Rat oracles test finite float decoding and numeric
meaning; existing numeric contract tests cover parsing, huge exponents, bounds
and consumers. Neither model alone proves the Go implementation.

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

- Immutable query stage against `af5e41179`: three alternating fresh-process
  pairs on the same 63-artifact design, Go 1.27.3, no concurrent task builds.
  Median allocation fell from 6,180,550,352 to 5,345,139,280 bytes (13.5%);
  elapsed time from 3.80 to 3.36 seconds (11.6%); CPU from 7.93 to 7.09 seconds.
  Peak RSS was 641,335,296 versus 647,610,368 bytes, essentially flat. All
  output trees matched exactly. Indexed sibling-query benchmarks at widths
  16/256/1024 allocated zero bytes per query; exhaustive reference traversal
  allocated 10,062/167,759/753,188 bytes respectively.
  The five representative designs were AsyncSessionSecurityDSL,
  JSONRPCCollectionBodyDefaultsDSL, PayloadBodyInlineRecursiveUserDSL,
  SelectedBodyDefaultsDSL and UnaryRPCsDSL. They cover async security, JSON-RPC
  collections/defaults, recursive inline bodies, selected body defaults and
  unary gRPC. Each matched base bytes across independent processes and compiled.
  Direct tests cover rejected handles, ambiguity, constraints and concurrent
  publication; the bounded [query model](../expr/tla/association_query/README.md)
  compares indexed lookup with exhaustive reachability.

- Compact exact-number stage against the immutable query stage: three alternating
  fresh-process runs per version on the same 63-artifact design and Go 1.27.3.
  Median allocation fell from 5,345,139,280 to 3,762,742,400 bytes (29.6%);
  elapsed time from 3.36 to 2.52 seconds (25.0%); CPU from 7.09 to 5.23 seconds.
  Peak RSS fell from 647,610,368 to 630,718,464 bytes (2.6%). All output trees
  matched exactly. The same five representative fixture designs matched across
  independent processes and their generated code compiled. Native integer
  conversion/comparison allocation tests require zero allocations; arbitrary
  precision remains available and exact. These figures measure generation,
  excluding Go compilation of its executable.

- Semantic memo lifetime stage against compact numbers: resolution, synthesis and
  projection tables now belong to their source, capture and plan owners. A
  separate forced-GC diagnostic after OpenAPI analysis reduced reachable heap
  from 186,674,904 to 106,903,896 bytes (42.7%). This is not a normal-run peak.
  Three alternating fresh-process pairs allocated 3,762,746,440 versus
  3,763,984,320 bytes (flat); median RSS was 609,370,112 versus 634,421,248
  bytes (4.1% higher), elapsed 2.84 versus 2.64 seconds, CPU 5.38 versus 5.33.
  These runs do not demonstrate a peak-RSS improvement. The ownership change
  removes a retention root; Go's collection timing and other live phases still
  determine peak memory. The 63 real-project artifacts and the same five fixture
  designs remained byte-identical across independent processes and compiled.

- Bounded artifact scheduling: three alternating fresh-process runs per setting
  on the 63-artifact project compared one, two and four workers. Median elapsed
  times were 3.44, 2.74 and 2.71 seconds; peak RSS 641,777,664, 614,481,920 and
  619,610,112 bytes. Allocations remained about 3.763 GB. Four is the measured
  default tradeoff, not a universal optimum. Against the preceding lifetime
  stage (2.64 seconds, 634,421,248 bytes), four workers were 2.7% slower and used
  2.3% less peak RSS in this run; these small differences do not establish a
  general speed or memory benefit. Bounded active work, deterministic errors and
  prompt release are the contract improvements. All 63 outputs and the same
  five representative designs were byte-identical and compiled.
