# Presence before selected-view validation

This model describes the conversion boundary fixed for #599. A decoded response
is converted to a pointer carrier before the selected view validates it. A field
required by the full result can be absent from a valid smaller view. Conversion
must preserve that absence so the selected-view validator owns acceptance.

## Design comparison

`legacy` uses full-type requiredness to skip guards: converting a missing inline
object or named scalar dereferences nil. Collections instead allocate empty
storage, losing absence before validation. `objects-only` guards dereferences
but still loses collection presence. `preserve` guards every nullable storage
location in the pointer carrier, independently of requiredness. This is the
chosen shared-transform rule; it needs no per-transport repair or default object.

The model enumerates four field kinds, full-type required/optional, selected or
omitted, known/unknown view, and supplied/absent. Conversion and validation are
separate transitions. Invariants check no panic, preservation of presence, and
acceptance exactly when the known selected contract is satisfied. Weak fairness
checks termination. The checked model has 192 distinct states.
The legacy transition models HTTP decode contexts, which do not apply source
defaults. Source-default contexts already guard objects/collections but used to
materialize an empty required array in the absence branch; the direct and
generated projection tests cover that additional path.

## Run and expected results

Use TLC 2.19, revision `5a47802` (8 August 2024), from the official TLA+ tools
release. The JAR SHA-256 is
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Runs used Corretto Java 25.0.4.1. From this directory, set `TLA_TOOLS_JAR` to
the installed JAR path and run each configuration:

```sh
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-presence-legacy -config legacy.cfg ProjectedPresence.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-presence-objects -config objects-only.cfg ProjectedPresence.tla
java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-presence-checked -config checked.cfg ProjectedPresence.tla
```

- `legacy.cfg`: exit 12, `NoPanic` counterexample for an absent required object.
- `objects-only.cfg`: exit 12, `PresencePreserved` counterexample for an absent
  required array becoming present.
- `checked.cfg`: exit 0; all invariants and termination pass.

Parse, evaluation, incomplete-state and deadlock errors are not successful
negative controls. Keep checker output outside the checkout and remove it after
validation and review; retain these sources and concise results.

## Implementation correspondence and limits

`codegen/go_transform.go` owns the implementation:

- `transformObjectPrimitiveInitExpression` guards named pointer conversions
  whenever the target also retains pointer presence.
- `shouldWrapTransformObjectField` preserves absent objects and collections for
  pointer targets; `wrapTransformObjectFieldCode` does not synthesize an empty
  required array in such targets.
- Existing value-target defaults and required-array encoding are unchanged.

`TestProjectedTransformPreservesRequiredPresence` checks emitted guards for
inline objects, named scalars, arrays and maps, with and without source defaults.
`TestResultViewPresenceGeneratedIntegration` compiles the real generated unary
HTTP and WebSocket clients and checks valid full/tiny/default selection, unknown
views, missing/null fields and missing nested required names. The baseline
panicked on tiny bodies and missing objects/scalars, and accepted missing
required arrays/maps. The corrected clients return validation errors instead.

The abstraction assumes native nil-capable field storage and supplied values
that satisfy their content constraints. It covers selection of fields whose
requiredness is inherited unchanged. It does not prove Go code generation,
recursive values, custom codecs, union representations, collection-element
nullability or view-specific requiredness overrides. Generated runtime tests
and existing customized-view/default/union tests cover production boundaries;
the model alone establishes none of those additional claims.
