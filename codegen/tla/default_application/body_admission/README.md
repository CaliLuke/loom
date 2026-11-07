# Admitted Body defaults before generic conversion

This unchanged finite model distinguishes raw observation, admission, selected Body default materialization and generic target conversion. It models the demonstrated Body9/payload7 bug, preserving the generic target-default rule. The nine configurations and TLA source retain the checked design inputs. The model proves only the finite phase and observation assumptions below.

Fourteen fixed inputs cover optional/required whole-body absence, required member absence, optional member absence with/without Body default, supplied zero/nonempty/empty collection, nullable/nonnullable null, selected/unmatched/ambiguous union state and a generic conversion. Values are symbolic tokens. RawAccepted is independent of mutated candidate state, and Expected derives from original input.

Assumptions: the codec has classified absence, supplied values and null; raw union matching has supplied branch-selection facts. Only one selected body's field is represented. Defaults are valid finite values of compatible native storage. TargetConvert deliberately supplies payload7 for absence: a target without a default, pointer storage and deferred materialization are implementation controls, not model claims. Nested resolution/application, helper identity and closure, custom/fallible materialization, equality optimization, recursion and independent envelope bindings are outside the model. No parser, liveness or fairness proof is claimed.

The retained corrected batch has eight expected invariant counterexamples (exit 12) and owned success (exit 0, 58 generated/distinct states, empty queue, depth 5): current and target-first violate BodyDefaultWins; before-admission violates AdmissionBeforeDefaults; replace-supplied and replace-null violate PreserveSupplied; invent-body violates PreserveWholeAbsence; guess-union violates RejectUnadmitted; global-source-default violates GenericTargetDefault.

`guess-union` admits input while ignoring its precomputed selection state; it does not model defaults changing raw branch matching. The before-admission trace stops at `applied => admitted` at depth 2. A required absent member incorrectly admitted after defaulting is a consequence represented by the transition rules, not an additional retained trace. Do not cite either result more broadly.

The original attempt-1 exit 75 was an incomplete successor state caused by missing RHS parentheses around `(applied \/ doApply)`. It is excluded as semantic evidence. The corrected complete batch is authoritative; the design record retained exact source/tool inputs and excluded that failed attempt from the complete batch. The checked source/configurations are unchanged.

## Correspondence and limits

The implementation design's shared `DefaultApplication` chooses the source stage or generic target fallback. Raw requiredness and union admission precede it. Physical presence owns absence; existing renderer/storage owners materialize and convert. These phases match the symbolic model. Its tokens do not prove that a native scalar no-op optimization is valid, or that the renderer fills nested defaults correctly; direct tests must establish those obligations. In particular, equal raw object defaults do not prove equal completed materialization.

The adjacent [conversion and storage models](../README.md) describe generic target defaults and declared storage. This model supplies their admitted Body phase boundary. Formal tools stay explicit design aids, never default tests, hooks, CI or release prerequisites.

The separate [discovery/copy exploration](DISCOVERY.md) covers the later observed
mapped-child and nullable-root identity loss, fixed graph ownership and helper
allocation. Its results do not extend this admission model's assumptions.

## Implementation correspondence

- `TransformAttrs.DefaultForField` / `DefaultApplication` own the selected absent-branch operation. `AttributeExpr.GetDefault` on the mapped Body occurrence supplies its effective selected default; the renderer retains its `FieldFacts` winning-member lookup.
- `transformObjectDefaultValueCode`, `transformOptionalField`, `transformNativeToOptional` and `transformNullablePresence` consume the shared operation. Physical presence owns absence; native zero, nonnil empty values and explicit nullable null remain supplied.
- `(*transformFailures).discoverObject` inspects the same operation for materialization and conversion fallibility. `transformFailurePlan` is the graph-building function, not the method receiver. Reachable changed actions inform helper identity through the existing graph and NameScope ownership.
- `payloadBuilder.buildTransformCode` opts in only admitted incoming Body conversion. The server's optional-whole-object guard must enclose that operation; bound fields remain independent.
- Generic `init_union_fields.go` also calls `transformFailurePlan` without opting in. gRPC's `applyObjectFieldDefaults` and `protobufRepresentationCanFail` remain generic target-owned consumers. Ordinary HTTP/JSON-RPC, response and CLI transforms do not acquire source preference globally.

Maintained public correspondence lives in `TestBodyTemplateDefaultContracts` and `TestBodyTemplateInheritanceGeneratedHTTP` with `BodyTemplateInheritanceDSL`: original Body9/payload7, defaultless pointer target, nested/whole-object materialization, required absence and supplied zero/empty/null. `TestTransformDefaultUsesTargetOccurrence` remains the generic source7/target9 control. The implementation qualification must additionally bind selected-union, custom/fallible and whole-body presence controls to the actual candidate; this small model does not prove them.

## Opt-in reproduction

Use Corretto 25.0.4.1 and official TLA+ tools 1.7.4 (TLC 2.19 revision 5a47802). Set `TLA_JAR` to the installed JAR. Its expected SHA256 is `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`. Verify the JAR and Java version before execution; an arbitrary PATH Java is not the original tool binding.

From this directory:

```sh
: "${TLA_JAR:?set TLA_JAR to the pinned tla2tools JAR}"
java -version
shasum -a 256 "$TLA_JAR"
model_state_dir=$(mktemp -d "${TMPDIR:-/tmp}/loom-body-default-owned.XXXXXX")
java -XX:+UseParallelGC -Xmx256m -cp "$TLA_JAR" tlc2.TLC \
  -workers 1 -metadir "$model_state_dir" -config owned.cfg BodyDefaultAdmission.tla
```

Use a fresh state directory for each configuration; run the eight negative configs individually and inspect the named invariant and exit 12. Parsing/evaluation failures or timeouts do not count as successful negatives. Retain exact inputs and concise results; remove only the owned checker state directories after validation and independent review, preserving reusable tools/caches. The original bounded supervisor used 30 seconds per process. No proof execution is added to CI.
