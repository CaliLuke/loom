# Error data ownership

This bounded model informed #590. It compares mutable merging, copying only
the result, copying result/history structs without nested metadata, and owning
all exposed snapshot data. The model represents two contributions, one merge,
one history read, and mutations of messages, field values and remedies.

`InputsRetained` requires merging and modifying result metadata to leave the
first operand unchanged. `HistoryRetained` requires mutations through a returned
history to leave its source unchanged. Shared pointers are modeled by writing
both aliases in one transition. Field and remedy are abstract scalar cells;
the model does not cover Go allocation, arbitrary cause graphs, concurrent
application writes, nil operands or repeated merges. Those are explicit test
or contract boundaries, not properties proved by this model.

| Configuration | Result | Distinct states | Depth |
| --- | --- | ---: | ---: |
| Legacy | `InputsRetained` counterexample, exit 12 | 2 | 2 |
| MergeOnly | `HistoryRetained` counterexample, exit 12 | 4 | 4 |
| Shallow | `HistoryRetained` counterexample, exit 12 | 5 | 4 |
| Owned | Both invariants pass, exit 0 | 12 | 5 |

The supported design snapshots data at merge, history attachment, and history
read boundaries. Nil remains the algebraic identity, preserving arbitrary error
types. Cause references deliberately keep identity; arbitrary errors cannot be
deep copied. A separate immutable API and an opt-in mode were rejected because
they leave ambiguous ownership in the default shared operation.

Implementation: `MergeErrors`, `History`, `WithErrorHistory`, and
`cloneServiceErrorEntry` in [error.go](../../error.go). Before the change,
[ownership regressions](../../error_ownership_test.go) reproduced input mutation,
history mutation and lost wrapped-cause identity. They cover repeated merges,
duplicates, nil identity, metadata copies and flattened attached history.
[The protobuf round trip](../../../grpc/error_ownership_test.go) checks original
messages/fields and snapshot ownership after wire reconstruction. Generator
output is unchanged; generated validation already assigns the merge result.
An audit of 15,730 matched generated/handwritten call lines in nine local
consumers found no candidate ignored return values; this is adoption evidence,
not an exhaustive public-consumer audit or an AST proof.

Use the official TLA+ tools v1.7.4 JAR (TLC 2.19, revision `5a47802`), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Runs used Corretto 25.0.4.1 and Go 1.27.1. From this directory, repeat for
`Legacy`, `MergeOnly`, `Shallow`, and `Owned`:

```sh
java -XX:+UseParallelGC -Xmx256m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-error-ownership -config Owned.cfg ErrorOwnership.tla
```

From the repository root:

```sh
go test -race ./pkg ./grpc -count=1
```

Keep checker output outside the repository and remove it after independent
review. The original three configurations must fail the named invariant, not
merely exit nonzero because the checker cannot run.
