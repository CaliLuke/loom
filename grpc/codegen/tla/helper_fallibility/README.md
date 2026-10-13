# Helper-local protobuf conversion fallibility (#754)

One generated helper is shared by a source/target conversion pair. Its error
parameter must depend on that conversion, not on whichever root first emitted
the helper. A fallible root may call a pure helper without an error argument.
Every helper that may fail, directly or through its reachable helpers, must have
an error channel. Recursive calls obey the same contract.

The parent generator violated this identity: `buildObjectHelper` records the
root's errorAware flag, `renderTransformHelperCall` passes the caller's flag,
and `appendTransformHelpers` deduplicates by name. The original #754 union/Any
witness and the #740 review's ordinary sibling-Any witness expose the same bug.

## Model

The model varies acyclic, self-recursive and mutually recursive helper graphs,
local conversion errors at every node, and independent sibling errors in two
root callers. Reachability establishes transitive helper fallibility. The first
root emits the shared declarations; both roots and all nested callers must use
the declared signatures. Independent sibling booleans cover either declaration
order. The local design passes 192 states; four alternatives fail:

- `root-inherit`: current root-owned signatures disagree between roots.
- `declarations-only`: local declarations still disagree with inherited calls.
- `calls-only`: local calls disagree with inherited declarations.
- `drop-errors`: consistent pure signatures lose actual conversion errors.

Run sequentially from this directory with Corretto Java 25.0.4.1 and TLC 1.7.4
(TLC2 2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
for config in root-inherit declarations-only declarations-only-nested calls-only drop-errors local; do
  mkdir -p "/tmp/loom-754-java-$config"
  java -Djava.io.tmpdir="/tmp/loom-754-java-$config" -XX:+UseParallelGC \
    -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-754-model-$config" -config "$config.cfg" HelperFallibility.tla
done
```

The five negative configurations must exit 12 with an ABI or error-propagation invariant
violation, not a parser/tool failure. Local must exit 0. The extra declarations-only-nested configuration checks the
nested call invariant independently: local declaration updates cannot leave a
fallible parent passing its own error argument to a pure child.

## Implementation correspondence and limits

`conversionCanFail` computes conversion fallibility from a source/target pair.
Root initialization, helper declaration/body context and helper call arguments
consume that same rule for their own pair. Copy the caller's transform context
before selecting helper-local fallibility; an emitted child must not mutate its
parent. Deduplication continues to share identical pair helpers, independently
of caller context. Root public constructor fallibility remains unchanged in #754:
Any conversion already returned errors. No names encode caller-specific modes.

`TestHelperFallibilityReadiness` generates both declaration orders for sibling
Any, union Any, recursive union Any and nested Any; all eight fail compilation
on the parent, while two all-pure controls build, vet and run.
`TestProtoHelperErrorContract` checks detached contexts and exact call/body
agreement; `TestConversionFallibility` covers reachable aliases and recursion.
`TestHelperFallibilityRuntime` checks malformed JSON and invalid protobuf strings
through nested helpers, unary encoders and both conversion directions, plus valid
protobuf wire round trips. `TestHelperFallibilityAdapters` checks stream send and
declared-error conversion propagation. Generated artifact comparisons cover both
orders, recursive unions, existing Any paths and pure controls.
This model assumes helper identity, graph discovery and source/target matching
are correct; it does not establish generated Go compilation or codec semantics.

#740 is separate. It adds local element-admission failure to the conversion
contract after #754 is delivered. A plain Leaf helper remains pure even when an
array occurrence checks nil before calling it. A helper whose own body contains
such arrays becomes fallible through that body's operations. Nil-element
admission must not move into a reusable optional-object-field helper. The
element admission model (tracked separately in #740) establishes that
separate occurrence boundary; neither model alone proves both obligations.
