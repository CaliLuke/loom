# Scalar field query ownership

FieldFacts models capture of scalar field meaning once, read-only queries and
later caller mutation. Owned checks snapshot meaning and bounded analysis work;
Borrowed reads the mutated source and Copied charges a descendant-graph copy per
query. EagerCopied models the former transform matching path, which copied the
descendant graph during construction before restoring the original child edges.
It abstracts the scalar table construction as one immediate-field pass:
actual AllRequired may have nonlinear normalization cost, which this model does
not prove or count. It assumes correct normalization/default/alias semantics;
Go parity tests cover those details. GC timing and arbitrary caller races are
outside the model. Construction requires a stable graph, while published scalar
facts are immutable and have no pointers back into that graph.

Use TLA+ tools 1.7.4 (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
and Corretto Java 25.0.4.1. Set TLA_JAR to the pinned jar. From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Owned.cfg FieldFacts.tla
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Borrowed.cfg FieldFacts.tla
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Copied.cfg FieldFacts.tla
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config EagerCopied.cfg FieldFacts.tla
```

Owned passed with 81 generated/32 distinct states, depth 6. Borrowed violates
SnapshotMeaning with 7 generated/7 distinct states, depth 3; Copied violates
NoRepeatedGraphWork with 3/3 states, depth 2. Negative controls must fail.
EagerCopied violates NoRepeatedGraphWork in its initial state. EagerCopied and
Owned were checked again with the pinned jar and Homebrew Java 26.0.1 for #1071;
Owned still passed with 81 generated/32 distinct states, depth 6.

The implementation owner is expr.ObjectFields; its fields contain scalar values,
winning original indices/keys and logical order, not retained AttributeExprs.
ObjectFieldFacts tests cover mapped equivalence, duplicate selection, mutation,
recursive-graph collection and allocation-free queries. The codegen required-field
regression previously grew from 1,177 to 13,201 allocations when an unrelated
nested object widened from 16 to 512 fields. This directly checks the measured
copying boundary beyond the abstract work counter.

Codegen's private transformFields borrows selected children from the stable
transform-owned graph and uses ObjectFields for normalized order, requiredness,
defaults and declaration layout. Its lifetime is one walk, without a global
pointer cache. Discovery and emission retain the same actual child occurrences;
callers must not mutate the graph during either pass. The model proves only
scalar snapshot meaning and abstract work, not borrowed child immutability or
Go allocation bounds. TestTransformFieldsPreserveMappedSelections checks exact
child identity, duplicate selection, defaults and source immutability;
TestTransformFieldsKeepDeclarationStorage checks occurrence versus declaration
requirements. TestTransformFieldsWinningExactKeyDefault checks that default
selection stays on the winning physical child when an exact key is repeated.
TestTransformFieldQueriesDoNotCopyDescendants previously allocated
285 versus 3,293 objects at descendant widths 16 and 512; the new path allocates
55 in both cases. The direct transform suite also covers nullable/presence,
default application, recursive helper discovery and rejected conversions.
