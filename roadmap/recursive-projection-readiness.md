# Recursive projection readiness — #788

Status: readiness independently approved; implementation released by root; implementation and bounded validation complete; frozen for root integration and exact-diff review.

Parent: `4535a403048510f055fc59ec3dfed41e3769ad58`.
Worktree: `/Users/luca/.codex/worktrees/recursive-projection/loom`.
Owner: projection agent. Root owns orchestration and integration.

## Scope and contract

One atomic structural ticket, #788, owns recursive selected-view projection. #730
(adjacent field mutation) and #779 (union disclosure) are linked regression
witnesses of incomplete projection ownership. Root explicitly excluded #784:
its HTTP client/server union registry has a separate physical declaration owner.
A durable `excluded_union.go.txt` repro remains beside the projection model. #727 is excluded: SSE selects the wrong
body for an already-correct runtime view, a separate transport selection task.

Every occurrence of a result type selects its authored nested view or its default
view, independent of containment in objects, named wrappers, arrays, maps and
union branches. Projection traverses supported recursive object references and
preserves occurrence identity. Changing excluded fields cannot change projected
values or emitted wire bytes. Reverse conversion reads only selected fields,
including when the canonical type marks an excluded field required. Authored
union cycles remain rejected; this work does not expand recursion support.

The documented consumer contract is `docs/code-generation.md` result views.
Real consuming contracts are the existing multiple/recursive result-view DSLs,
HTTP result-presence fixture, JSON-RPC viewed unary/stream fixtures, and gRPC
viewed-result fixtures. Application code still chooses its view and populates
selected required fields; it does not redact nested service values manually.

## Observed owners and candidate repair

1. `expr/value_plan_views.go` owns `expr.Project`, but `projectType` sends maps
   and unions to `plainType`, preserving their canonical children. Its object
   fallback likewise bypasses nested result projection. Complete recursive
   semantic selection here, maintaining memoized identity and cycle handling.
2. `codegen/service/service_data_views_init.go:buildConstructorCode` removes
   direct result fields while iterating and sends the remainder to generic
   `GoTransform`. This both skips adjacent fields and forgets selection through
   collections/unions. Replace the split with one recursive conversion plan,
   consuming expression selection and retaining physical carrier references.
3. `codegen/service/service_data_views.go` collects all physical view carriers.
   A physical carrier can support several selected views; selection must not
   mutate or prune this shared declaration. Keep semantic selection separate
   from carrier layout, but make conversion and validation use the same selected
   graph. Helper identity must include selected view, not only Go type names.
4. HTTP/JSON-RPC body builders already consume expression projection; no
   transport redaction was added. The distinct HTTP `UnionTypes` registry emits
   declarations into both client and server packages despite differing physical
   body graphs. That residual #784 owner is excluded from this implementation.

No per-transport secret filtering or generated-code postprocessing is allowed.
The concrete generic seam is an optional `TransformOptions` argument to
`codegen.GoTransform`, with `HelperName func(source, target *expr.AttributeExpr)
string`. `TransformAttrs` carries it and the single `transformHelperName`
function consults it for both calls and declarations. Existing callers omit
options. The service-owned conversion plan binds detached semantic subsets to
canonical physical carriers and allocates names by semantic source/target
identity, including selected view. The callback chooses identity only; generic
code owns recursion and rendering and receives no view policy. No callback
bodies, fake Go type names or duplicate renderer are introduced. Add a direct
`codegen/go_transform_helper_rule_test.go` case proving distinct semantic
occurrences with the same physical Go types have distinct helpers and that every
call has one declaration. Readiness review must approve this concrete seam.

Helper allocation uses a service-owned registry keyed by the complete selected
source/target occurrence identity and conversion direction. Selected result IDs
retain their view, and ordinary wrapper identities include their selected child
graph. The plan annotates its detached copies with stable identity so
`NewMappedAttributeExpr`/`DupAtt` cannot erase it; pointer-address identity cannot
serve as the callback key. It allocates helper names on first traversal in deterministic graph
order with a conversion-local NameScope, reuses each allocation for both emission and
`collectHelpers`/`generateHelper` memoization, and does not range maps for output.
Supported recursive references point back to the same selected graph node, so
recursion is cut by semantic identity rather than only physical Go type name.
The exact identifier storage stays private to the service conversion plan; the
generic transformer only receives an allocated name. Independent-process output
comparison checks this correspondence rather than assuming map iteration order.

The value-contract correspondence ledger explicitly treats DSL-to-plan
extraction and Go graph traversal as tested adapter boundaries, not universal
Lean theorems (`expr/lean/value_projection/correspondence.md`, table row for
Plan/Field/Branch and naming/nullable-view controls). This change repairs that
unproved extraction boundary; it does not change canonical value resolution,
scalar codecs or Lean statements. Record the newly exposed map/union
counterexamples and completed direct/generated correspondence in that ledger.
The new bounded projection model extends coverage beyond the existing
`result_output` model, which assumes projection is correct and covers boundary
ordering/classification only. Existing `view_union_names`, `union_declarations`
and `codegen/tla/projected_presence` claims must retain their stated scopes;
rerun their focused checks where conversion/declaration changes touch them.

## Ordered tasks and proposed files

- [x] Add regression witnesses before implementation in
  `expr/result_projection_recursive_test.go`,
  `codegen/service/recursive_projection_test.go`, and shared fixture
  `codegen/service/testdata/recursive_projection_dsl.go`. Extend existing view
  fixture definitions where their shapes provide the needed witness.
- [x] Add model source/configurations and a correspondence/evidence README in
  `codegen/service/tla/recursive_projection/`.
- [x] Complete the expression selection owner in `expr/value_plan_views.go`.
- [x] Replace direct-field partitioning in
  `codegen/service/service_data_views_init.go`; place an owning conversion plan
  in `codegen/service/projection_conversion.go` if needed. Audit
  `service_data_views.go` and `service_data_views_validation.go` for collection
  and union selection parity. The reviewed helper-identity option changes `codegen/go_transform.go`,
  `codegen/go_transform_collections.go` and `codegen/transformer.go`.
- [x] Investigate HTTP union reachability; preserve and exclude separate #784.
- [x] Add generated regressions through existing module renderers in
  `http/codegen/recursive_projection_test.go`,
  `grpc/codegen/recursive_projection_test.go`, and
  `jsonrpc/codegen/recursive_projection_test.go`. Reuse existing render/run
  helpers; no replacement generation harness.
- [x] Update `docs/code-generation.md`; send framework/consumer skill deltas to
  root, which owns shared skill and roadmap updates.
- [ ] Independently review the exact final diff after checks; resolve all
  findings, freeze uncommitted candidate for root integration and fresh exact-diff review.

## Bounded acceptance matrix

| Case | Obligation | Evidence |
| --- | --- | --- |
| Adjacent direct result and CollectionOf fields with trailing ordinary field | No skipped conversion, nested tiny excludes canonical required field, both directions | Direct rendered helper assertions and generated client round trip |
| ArrayOf/MapOf result elements; map-to-array nesting | Default result selection on every path; nil/empty retained | expr structural selection, direct conversions, HTTP/gRPC wire |
| Union result branch plus array of same result type, both declaration orders | Shared carrier cannot widen projection; body/declaration consistency | expr graph, generated HTTP/JSON-RPC bytes and compile |
| Union field excluded by all selected nested views | Separate physical registry defect #784 | Preserved repro; explicitly excluded from acceptance |
| Explicit nested tiny/default views of same type in one parent | Helpers distinguish selected view while sharing physical type | Direct helper identity and generated values |
| Existing recursive object and recursive collection DSLs | Termination and supported recursion preserved | Existing TestViews/TestService cases, generated compile |
| Existing nullable/optional fields and required unions | Physical presence and union layout unchanged | Existing presence integration and service tests |
| Excluded-field perturbation on unary and one stream per transport | Excluded data cannot affect bytes, selected values retained | Generated HTTP SSE, gRPC server stream, JSON-RPC SSE |
| Authored union cycle | Existing design rejection remains | Existing expr/dsl regression; no new supported contract |

No default/presence semantic migration, generated-name redesign, schema numeric
work, stream lifecycle changes, or unrelated validation fixes belong here.

## Model before implementation

`RecursiveProjection.tla` will model occurrences identified by shape, selected
view and canonical type. Bounded shapes include direct object, adjacent object,
array, map, union and recursive reference. A legacy mode admits canonical fields
on non-direct paths or skips adjacent fields; a checked mode selects at each
result occurrence. Invariants: excluded-field noninterference, no conversion
read of an excluded required field, declaration references agree with selection,
and order-independent projection. Values, Go memory cycles and arbitrary codecs
are outside this structural model; compiled transport tests establish mapping.

Run from `codegen/service/tla/recursive_projection`, using the installed TLC 2.19
JAR pinned by the neighboring `result_output/README.md` (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-recursive-projection-legacy -config legacy.cfg RecursiveProjection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -metadir /tmp/loom-recursive-projection-checked -config checked.cfg RecursiveProjection.tla
```

Legacy must fail a named invariant with a real counterexample; checked must pass.
The cached checker is `/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`;
the authentication agent verified its SHA-256 against this pin and Java Corretto
25.0.4.1. Its filename is historical; checker identity comes from the hash.

## Validation and output comparison

Before changes, run new direct/generated tests at the baseline and record exact
failure messages. Run implementation checks with
`LOOM_DIR=/Users/luca/.codex/worktrees/recursive-projection/loom`.

```sh
go test ./expr -run 'TestRecursiveResultProjection' -count=1
go test ./codegen/service -run 'TestRecursiveProjection|TestViews|TestProjectionParity' -count=1
go test ./codegen/service -count=1
go test ./http/codegen -run 'TestRecursiveProjection|TestCustomizedResultViewsGeneratedModule|TestResultViewPresenceGeneratedIntegration' -count=1
go test ./codegen -run 'TestGoTransform' -count=1
go test ./grpc/codegen -run '^(TestRecursiveProjection.*|TestGRPCProjectionParity)$' -count=1
go test ./grpc/codegen -run '^TestResponseEncoder$/^response-encoder-result-with-(views|explicit-view)$' -count=1
go test ./grpc/codegen -run '^TestStreaming$/^server-streaming-result-(with-views|collection-with-explicit-views)$' -count=1
go test ./jsonrpc/codegen -run 'TestRecursiveProjection|TestJSONRPCViewedUnaryResultGeneratedModule|TestJSONRPCViewedStreamResultGeneratedModule' -count=1
git diff --check
```

The new module tests compile, vet and execute generated code with real runtime
encoders. The existing compile-corpus harness deletes its temporary modules;
`LOOM_TESTDATA_RESULTS` retains observations only, so do not use it to capture
comparison files. Use the existing CLI against explicit task-owned modules in
`/tmp/loom-recursive-projection-comparison/{base,current,repeat}/CASE`, with the
same `design/design.go` wrapper calling a selected exported DSL. Each module
requires the pushed baseline SHA and has a local replacement to the relevant
source worktree. All baseline commands use `LOOM_DIR=BASELINE_WORKTREE`; the
baseline must be a managed pristine worktree with Git metadata, because
`RepositoryRoot` needs it. Use the pristine read-only baseline at
`/Users/luca/.codex/worktrees/correctness-baseline/loom`; never copy witnesses
into that worktree.

The bounded ten cases are both declaration orders for each new `RecursiveProjectionTransportDSL` HTTP/gRPC/JSON-RPC regressions and existing
`ResultWithResultTypeDSL`, `ResultWithRecursiveResultTypeDSL`,
`ResultWithRecursiveCollectionOfResultTypeDSL`, and
`ResultWithOneOfInResultTypeDSL` from `codegen/service/testdata`. New regression
DSL source is copied unchanged to baseline/current/repeat design modules so the
baseline need not export it. For each exact module, using its selected source:

```sh
LOOM_DIR="$SOURCE" go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen example.com/projectioncomparison/design
LOOM_DIR="$SOURCE" go run -mod=mod github.com/CaliLuke/loom/cmd/loom example example.com/projectioncomparison/design
go mod tidy
go build ./...
go vet ./...
```

`base` uses baseline source, `current` and `repeat` use implementing source;
current/repeat are separate fresh CLI processes. Compare generated directories
with `diff -ru BASE/CASE/gen CURRENT/CASE/gen` and
`diff -ru CURRENT/CASE/gen REPEAT/CASE/gen`, excluding nothing from generated
output. Manifest source-version differences must be explicitly reported if
present; they are not ignored silently. Compile every selected module; baseline
compile failures are expected only on the new bug witness. The directories stay
available for independent review, then are manually removed.

Intended differences: view-aware child conversions replace generic full-type
copies, reachable transport map/union child bodies use the selected fields,
and excluded fields remain unset. #784 unreachable union declarations remain
an explicitly excluded sibling. Existing valid
ordinary scalar/object output should remain byte-identical except documented
helper reshaping required by the single conversion owner. Record each actual
difference before review; do not accept unrelated golden churn.

After final validation/review, preserve concise source/command/results here or in
the owning model README and remove task-owned checker/output directories after
checking active users. Preserve shared caches and toolchains. Root records final
commit/push/integration status in the recovery tracker.


## Completion and handoff

Implementation and bounded checks are complete. See
[`codegen/service/tla/recursive_projection/README.md`](../codegen/service/tla/recursive_projection/README.md)
for exact commands, 30 successful build/vet receipts, 124-byte-identical generated
files across fresh processes, baseline/current runtime failures and repairs,
intended golden changes, model results and exclusions. This candidate is
uncommitted. Root owns skill edits, transfer, fresh exact-final-diff review and
the one #788 commit. Temporary evidence stays available until review finishes.
