# Protobuf helper fallibility — #754 prerequisite for #740

Status: implemented; awaiting independent final review and atomic delivery. Parent `fce66acb137cf7df0282bfe69446d9cceca1ceee`.
Worktree is the reused `default-application-parent`. #740 production remains parked.

## Sequence

1. [x] Confirm the #740 review blocker is the existing #754 report.
2. [x] Reproduce plain/Any, union and recursive mixed consumers in both orders;
   include nested fallible helper and all-pure controls.
3. [x] Model declaration/call/body agreement and transitive error propagation;
   compare root-inherited, partial-fix and helper-local candidates.
4. [x] Freeze shared-owner scope and bounded acceptance for independent review.
5. [x] Independent readiness approval; implement the shared helper-local owner.
6. [x] Bounded compilation, runtime, artifact comparison, determinism and lint.
7. [ ] Independent final review and one atomic commit.
8. [ ] Let #740 extend the delivered owner with occurrence admission separately.

## Failure classification

This is the existing #754 compile-time bug, independently reproduced at current
parent fce66acb1; it is not a new report. A helper shared by a plain root and a
root with unrelated Any content inherits whichever root emits it first. Its
source/target name is shared, while its call sites select error arguments from
each caller's root. The #740 review found the defect in an ordinary optional
Leaf field, so widening only root fallibility for #740 would reproduce it even
without Any.

`TestHelperFallibilityReadiness` adds ten generated modules: sibling Any, union
Any, recursive union Any, nested Any and a pure control, each in both method
orders. Eight mixed modules fail with too many/too few helper arguments in both
client and server packages. The two pure modules build/vet/run. The full bounded
probe finishes in 10.371s. Source and runtime toolchain are unchanged from #740:
Go 1.27.2, GOEXPERIMENT=jsonv2, explicit local LOOM_DIR. Exact log:
`/tmp/loom-754-current-parent-red.log`. The review's original two-case probe is
preserved separately under `/tmp/loom-740-review-helper-abi*`.

## Owner and atomic scope

Introduce one internal source/target conversion contract in grpc/codegen that
owns runtime fallibility. For #754 the predicate remains the existing reachable
Any behavior; no new nil rejection or public conversion failure is introduced.
`protoBufTransform` and `buildInitData` must derive root fallibility from that
owner. `buildObjectHelper` derives a detached helper-body context from the
helper's own source/target pair, and both its `TransformFunctionData.ErrorAware`
and nested calls consume that context. `renderTransformHelperCall` selects its
argument list from the called helper's contract, not the caller's fallibility.
`appendTransformHelpers` continues to share a helper by conversion identity:
its ABI is now invariant across callers and declaration order.

Do not append root-fallibility suffixes to helper names, duplicate pure helpers,
make every constructor return an error, or remove the error channel from genuine
Any conversions. Do not alter optional object nil behavior, wrapper semantics,
protobuf declarations, service types, codec mappings or transport classification.
The normal public constructors already have the correct root error signatures;
this repair makes their private shared helpers consistent. No new product policy
or user approval is needed. Document regeneration and contributor ownership.

## Formal design evidence

The helper-fallibility model covers acyclic, self-recursive and mutually recursive
helper graphs, all local error placements, and two callers with independently
fallible siblings. Transitive reachability determines each helper's capability.
Local ownership passes all ABI, propagation and exact-contract invariants over
192 states. Root inheritance, declaration-only, call-only and error-dropping
alternatives fail named invariants. The model does not claim to establish actual
Go signatures, source/target identity, Any codec behavior or generated type
selection; tests cover that correspondence.

## Bounded acceptance for implementation

1. Direct table-driven owner tests establish pure versus reachable Any pairs,
   alias/wrapper and recursive traversal. Helper declaration/body/call tests prove
   local agreement and preservation of caller context. A mixed chain must have
   fallible parent -> pure child and fallible parent -> fallible child calls.
2. All ten existing readiness modules must build/vet/run after the repair. Keep
   both declaration orders and recursive calls; adapt the probes into durable
   regression tests, not passing assertions of current failure.
3. Generated runtime tests must return malformed/unsupported conversion errors
   through constructors, unary encoders, stream sends and declared error details;
   reuse their harnesses. Existing CLI protojson, constructor/encoder structural
   and compiled Any round-trip controls remain green. Normal Any/null and
   all-pure controls remain unchanged. Receive conversion errors are exercised
   through the same generated constructors; no new streaming timing claim is made.
4. Compare bounded rendered artifacts for sibling-Any both orders, recursive
   union-Any, existing AnyErrorDSL, mapped-error/stream adapters and one pure control at parent/candidate/repeat.
   Parent mixed builds are expected failures (record exact diagnostics), while
   candidate builds and vets pass. Independent repeat must match exact bytes.
   Intended changes are only helper error parameters and corresponding calls
   and body contexts. Schema, service types, public root signatures and valid
   wire values stay unchanged. Record every changed artifact and relevant goldens.
5. Run affected direct/package gates and independent final review. Deliver #754
   as exactly one atomic commit before #740 implementation begins.

#740 consumes this owner by adding occurrence admission as a possible local
failure. A plain object helper does not become fallible merely because a parent
array can contain nil; the array converter owns that failure. A recursive Node
helper containing object arrays is itself fallible. Revised #740 acceptance adds
both-order mixed callers for these two cases. Its 1296-state admission model
remains valid and explicitly separate from this helper-signature obligation.

## Final implementation and evidence

Four production files change: `protobuf_transform.go` owns `conversionCanFail`;
`service_data_convert.go` consumes it for roots;
`protobuf_transform_helpers.go` detaches helper contexts and selects local
fallibility; `protobuf_transform_collections.go` selects the called helper's ABI.
No nil admission, public signature, schema or wrapper policy changes are included.
Readiness review approved the model and owner before these edits.

The direct helper regression failed before implementation (four context ownership
failures). The ten mixed/pure modules now pass. Focused transformation, Any and
CLI tests pass (41.171s); affected client/server goldens, recursive type, map-branch,
union, default and scalar-presence controls pass (39.833s). The nested-helper
runtime module passes (4.496s), and the mapped-error/stream runtime module passes
(4.167s); generated runtime probes run with the race detector. Package golangci-lint
reports zero issues; size, formatting, JSON-v2 and NameScope gates pass (existing
file-size warnings remain). No checked-in gRPC `gen/` fixtures exist.

Six specimens produce 66 generated files. All candidate/repeat bytes match.
The only eight changed artifacts are client and server `types.go` for
`plain-first`, `fallible-first`, `recursive` and `adapters`. Pure Leaf calls omit
an irrelevant error argument; helpers first emitted by fallible roots also omit
that parameter from their declarations. All protobuf, service, public constructor,
encoder and stream-adapter files are unchanged. Parent mixed specimens fail with
known too-many/too-few argument diagnostics; other parent controls compile.
All candidate specimens build and vet. No broad exported-design audit was run.

Exact evidence is frozen with `/tmp/loom-754-final-manifest.json` and
`/tmp/loom-754-final.patch`; generated hashes and every output difference are in
`/tmp/loom-754-artifact-comparison.json` and `/tmp/loom-754-generated.diff`.
Model source/configuration hashes are unchanged from approved readiness, so its
192-state positive run and five negative counterexamples remain valid evidence.
The final packet excludes all parked #740 readiness files. #740 must consume this
delivered owner and obtain its own readiness approval before adding failures.

Reproduce the targeted runtime and owner checks with local source:

```sh
LOOM_DIR="$PWD" GOEXPERIMENT=jsonv2 go test ./grpc/codegen \
  -run '^Test(ProtoHelperErrorContract|ConversionFallibility|HelperFallibility(Readiness|Runtime|Adapters))$' -count=1
LOOM_DIR="$PWD" LOOM_HELPER_FALLIBILITY_OUTPUT=/tmp/loom-helper-artifacts \
  GOEXPERIMENT=jsonv2 go test ./grpc/codegen \
  -run '^TestHelperFallibilityArtifact(s|Builds)$' -count=1
```

Generate artifacts again in a separate process and directory for determinism;
compare all `*/gen/**` bytes. The model README records the checker commands,
invariants, counterexamples and limitations.
