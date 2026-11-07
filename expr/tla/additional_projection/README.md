# Typed additional-member representation

Typed additional members are edges of the captured value graph, just like
object fields and collection values. Schema projection, examples, schema
matching, runtime carrier decoding and generated matching descriptions must
follow the same edge. A byte value beneath it uses JSON base64 and decoded-byte
length constraints; it cannot inherit the baseline binary-string schema.

The old plan omitted additional edges. The OpenAPI analyzer then passed absent
plan positions, its byte identity walk skipped the edge, and baseline records
had no corresponding child. Value observation also serialized raw extras rather
than applying their typed plan. Fixing one renderer would leave those owners in
conflict. The corrected plan captures the additional child and exact required
and omitted keys, records source ancestry, and all consumers descend through it.
The HTTP preparation and graph-copy walks also include the edge, independently
of untagged-root versus ordinary nested decoder context.

The bounded model has four nodes, including a recursive additional edge, three
consumers and two key classes (empty and ordinary). Capture processes each node
once; all consumers must use the complete graph. Presence is independent of key
spelling. Four occurrence roles (base, x, y and an alias of x) must retain their
exact required-key sets without sharing local predicates. It abstracts Go
pointers, codec details, serialization and
validation predicates; tests establish those implementation properties.

| Configuration | Result |
| --- | --- |
| `drop.cfg` | `Complete` fails when the additional edge is omitted |
| `schema-only.cfg` | `SharedRepresentation` fails when only the schema consumes the edge |
| `name-sentinel.cfg` | `KeyPresence` fails when the empty key signals absence |
| `owner.cfg` | All invariants pass: 18 generated, 14 distinct states, depth 6 |

The empty-key counterexample corresponds to `resolvedExtra`: the previous
`name != ""` guard silently discarded a supplied empty key. It now records
presence explicitly, so generated examples and value projection retain the key.

Run explicitly with Java 25 and TLA+ tools 1.7.4 (TLC 2.19, jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
for mode in drop schema-only name-sentinel owner; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-additional-projection-$mode" \
    -config "expr/tla/additional_projection/$mode.cfg" \
    expr/tla/additional_projection/AdditionalProjection.tla
done
```

For the alias controls, copy `owner.cfg` to a temporary file and replace
`Policy = "owner"` with `Policy = "alias-drop"` or `Policy = "alias-mutate"`.
Run TLC with that temporary configuration and a separate metadir. Both exit 12
with `OccurrenceRequirements`: the former loses alias-local requirements; the
latter leaks them across the shared declaration. The corrected owner retains the
18 generated, 14 distinct states, depth 6 result.

The three original controls exit 12 for the listed invariant; owner exits 0. These checks
are opt-in design aids, never CI or release gates. Delete temporary checker
state after independent review; keep the model and summarized evidence.

## Implementation and regressions

- `expr/value_plan_build.go` captures additional edges; `value_plan_schema.go`
  exposes them and `value_plan_query.go` retains source-parent reachability.
- `expr/value_projection_{build,decode,schema}.go` applies typed plans, while
  `value_projection_additional.go` owns exact required/forbidden membership.
- `expr/value_plan_json_shape.go`, `pkg/json_shape.go`, and
  `codegen/unionjson/{union,shape}.go` retain that policy in generated matchers.
- HTTP `representation` records every additional container's runtime policy and
  copies declarations separately for the decoder contexts that own them.
- OpenAPI `analyzer_baseline.go` records the additional schema slot;
  `analyzer_representation.go` includes it in byte projection identity;
  `analyzer.go` supplies the structural and example child positions.

`TestTypedAdditionalValueProjection` covers bytes, arrays, maps, objects, tagged
unions and nullable values with schema and carrier checks. Recursive, omitted
and literal-key tests check graph identity and key semantics.
`TestTypedAdditionalOccurrenceRequirements` compares base, two siblings, alias
chains and nested array/map occurrences through source resolution, schema
matching, JSONShape and carrier decoding. Required keys are captured even when
the additional edge is on a named declaration; alias nodes apply their own
membership predicates without mutating their shared child. Direct HTTP and
unionjson tests cover decoder contexts, hidden-member policy and emitted
matching graphs. Baseline tests prove JSON/raw separation and nonmutation.

The maintained `AdditionalCompositeDSL` and JSON-RPC variant include named
constrained bytes, nested arrays/maps, nullable required extras, composite-only
reachability, recursion, mapped fields and literal names. Generated HTTP/RPC
client/server tests compile and exercise success and rejection paths. OpenAPI
3.1/3.2 JSON/YAML tests use libopenapi and wire assertions; the specimens pass
pinned Vacuum 0.36.2 contract rules. Regenerate OpenAPI documents to pick up
correct additional-byte schemas and examples; wire formats are unchanged.

## Bounded output acceptance

The comparison uses the same new specimen inputs on the pre-change production
owners and the corrected owners, with other concurrent source changes snapshotted
identically on both sides. Four maintained designs cover HTTP/RPC composite
extras, ordinary typed additional attributes, and ordinary untagged arrays.
There are 168 raw/rendered sections: 156 byte-identical and 12 changed, with no
added or removed section. The intended differences are only:

- Composite OpenAPI JSON/YAML byte schemas use base64 and projected byte-length
  predicates, including named, nullable and nested collection occurrences.
- Literal-key examples retain their required empty-string key in OpenAPI and
  both generated CLIs.
- Nullable byte-extra examples retain explicit null and appear in both CLIs.
- Examples missing alias-local required extra keys are suppressed in OpenAPI
  and CLI output instead of advertising invalid values. The maintained byte
  sibling specimen checks occurrence isolation in rendered reference siblings
  and generated HTTP/RPC acceptance and rejection paths.

Ordinary typed and untagged controls remain byte-identical. Both base and
candidate generated HTTP/RPC composite modules compile and pass the mounted
harness; the original defect is their schema/value projection disagreement.
Five independent candidate processes produce byte-identical output. Direct
regressions first fail on unvalidated extra values, wrong byte schema lengths,
and lost empty keys, then pass with the corrected owners.

## Decode traversal work

`DecodeWork.tla` models one chain of typed additional objects. The old decoder
validated a child, discarded that result, then decoded the same child again to
retain it. Eight nested objects therefore visited 511 values instead of nine.
The corrected decoder gathers extra keys, checks their admission, then decodes
and optionally retains each child in one traversal. Discarding an extra from the
returned value does not waive its typed validation.

Run explicitly with TLC 1.7.4 and separate temporary state directories:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config decode-discard.cfg -metadir "$TMPDIR/decode-discard" DecodeWork.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 -config decode-retain.cfg -metadir "$TMPDIR/decode-retain" DecodeWork.tla
```

The discard policy violates `SingleDecode` at depth one. Retaining the result
passes nine states through depth eight. The model counts decoder invocations;
it does not bound JSON parsing cost, branching input size, or allocation counts.
`TestAdditionalDecodeWork` guards against exponential allocation growth without
asserting wall-clock time or an exact compiler allocation count.
`TestAdditionalDecodeRetainsChildren` checks the retained typed values and rejects
an invalid nested leaf. These implementation tests remain ordinary test gates;
the model is an opt-in design aid.
