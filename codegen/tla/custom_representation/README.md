# Custom default representation and failure propagation

This bounded design model supports #926. A selected custom String field may be
supplied or absent. Supplied includes scalar zero and empty values; a separate
presence owner decides this fact. Only absence permits a default. Supplied values crossing physical representations
use the same semantic conversion owner without applying a default. An authored
parser may accept or reject it. A leaf conversion passes through two enclosing
conversion helpers before a result is published.

The current literal path assigns a native String expression to opaque custom
storage and violates `TypedAssignment`. A parser wrapper that panics violates
`NoPanic`; parsing correctly but losing the error in a parent violates
`FailureVisible`. Owned conversion propagation retains the error to the public
boundary and never publishes a failed partial result. It preserves supplied
values without parsing defaults, converts supplied values at representation
boundaries, and materializes accepted absent defaults.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| Literal | TypedAssignment counterexample | 9 / 9 |
| Panic | NoPanic counterexample | 9 / 9 |
| Drop | FailureVisible counterexample | 33 / 33 |
| Owned | All invariants and completion pass | 40 / 40 |

Checked with TLC 1.7.4, jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, Java 25.
From this directory, set `TLA_JAR` to the installed checker and run each config
with a distinct temporary state directory:

```sh
model_tmp=$(mktemp -d)
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir "$model_tmp/states" \
  -config Owned.cfg DefaultPropagation.tla
```

Repeat with Literal, Panic and Drop; each must fail the named invariant (exit
12), not parsing or resource errors. Preserve concise evidence and remove state
and trace files after review.

The model does not prove Go capability dispatch, authored parser correctness,
conversion-graph extraction, arbitrary recursion, view selection, or error text.
It assumes immutable inputs and correct presence/selection facts. Two parent
hops expose lost error propagation; they are not a proof for arbitrary graphs.
Implementation tests must cover direct, nested, collection and recursive helper
boundaries, parser failures, native and raw JSON controls, and selected views.
The direct pre-implementation tests are
`TestCustomStringValidationUsesSemanticRepresentation` and
`TestCustomStringDefaultUsesPhysicalRepresentation`. Published service-only UUID
compiler witnesses independently establish the implementation failure.

## Recursive helper classification

`FallibilityGraph.tla` distinguishes selected conversion pairs. AFull/BFull form
an A↔B cycle reaching a parser leaf; CFull is a second route to that leaf, so
propagation order can vary. ATiny/BTiny reuse the physical A/B declaration names
but exclude the leaf edge. NativeA/NativeB form a cycle without failures. A
Boolean default-enabled context toggles whether the leaf needs materialization.
An independent crossing context requires conversion of supplied values even when
defaults are disabled. Either operation seeds the same selected-pair graph.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| graph-Visited | CompleteClassification counterexample | 5014 / 1314 |
| graph-SharedKey | SelectedPairsOnly counterexample | 6 / 6 |
| graph-Fixed | Exact classification and completion pass | 35 / 26 |

Run with the same pinned checker, substituting e.g.
`-config graph-Fixed.cfg FallibilityGraph.tla`. The once-visited alternative can
cache A as infallible before B/C have been classified. The shared-key alternative
pollutes the tiny view merely because it shares physical names. Seed direct
failures on selected conversion pairs, then propagate dependency failures to a
fixed point. Required implementation tests must include both traversal orders,
recursive helper declarations/calls, excluded views and default-disabled targets.
This finite graph model assumes the extracted dependency graph is correct; it
does not prove graph extraction from arbitrary expression or physical type graphs.

## Protobuf graph extraction

The #937 regression exposed a violated extraction precondition: generic
object/union compatibility left a wrapper message paired with a service union.
`grpc/codegen.normalizedTransformAttrs` now shares the emitter's protobuf
compatibility rule and retains canonical child nodes when unwrapping. Helper
discovery and representation fallibility both consume it. The query keys
canonical attribute occurrences under fixed contexts; name resolution cannot
form a visited key because it reserves names for unwrapped unions that have no
message declaration. `TestProtobufPairAnalysisDoesNotAllocateMessageNames`
checks this purity requirement. The direct
`TestProtobufPairNormalizationPreservesCanonicalShapes` and
`TestProtobufUnionPairFallibility` cover both directions, collection and scalar
wrappers, mapped fields, recursive identity, native/custom leaves and rejection.
Existing exported union/collection fixtures check generated compilation. This is
a concrete extraction repair; the graph algorithm and modeled transitions are
unchanged, and the model still does not prove expression normalization.
