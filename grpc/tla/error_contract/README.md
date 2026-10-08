# Complete gRPC error contracts

This model informed #589. Independent error branches cannot describe each
other. The runtime must resolve one response contract before rendering status
and details, rather than running unrelated tree-wide searches for each.

The model enumerates two branches with Unknown, Canceled or Unavailable codes,
in every order, with or without an explicit outer owner. `CompleteFailure`
requires both contributions in aggregate details. `OrderIndependentStatus`
requires unanimous codes or Unknown. `ExplicitOwner` requires an authored
outer contract to win over its causes. The abstraction treats ordinary wrappers
as transparent and assumes branch codes have already been resolved. It does
not prove arbitrary Go error traversal, mapper execution, protobuf encoding,
message formatting, or concurrency safety. Finite, acyclic unwrap trees are
assumed, as with ordinary Go error traversal.

| Configuration | Result | Distinct states | Depth |
| --- | --- | ---: | ---: |
| Legacy | `CompleteFailure` counterexample, exit 12 | 19 | 2 |
| LegacyStatus | `OrderIndependentStatus` counterexample, exit 12 | 25 | 2 |
| Aggregate | All three invariants pass, exit 0 | 36 | 2 |

The supported policy keeps unanimous statuses, uses Unknown for conflicts, and
uses generic full-message fault details without inferring branch retry traits.
Picking the first branch was rejected because order would control callers'
retry decisions. Always discarding statuses was rejected because unanimous
branch contracts can be preserved. A direct outer ServiceError, GRPCStatus or
recognized designed mapping deliberately owns the complete failure. An outer
contract must not be confused with a named descendant inside one join branch.

Implementation correspondence:

- [classifyError](../../error_contract.go) owns direct-interface selection,
  single-cause traversal, branch consensus and designed mappings. Status and
  detail rendering both consume this result; no generator policy is added.
- [Runtime tests](../../error_contract_test.go) cover reordered and nested joins,
  nil/single errors, ordinary wrappers, uncomparable error values, nil/OK
  status providers, outer owners, mapping conversion failures, original status
  details and merged validation histories.
- [Generated-server tests](../../../codegen/generator/grpc_joined_error_test.go)
  reuse the client-boundary fixture, vet/build its generated code and exercise
  custom, wrapped, joined, reversed and unanimous failures through real unary
  and streaming gRPC calls over bufconn. Generator output is unchanged.

The runtime regressions first reproduced branch-selected status/details and
an inner status overriding an explicit outer ServiceError on the parent
revision. The model rejects the same incomplete selection before implementation.

Use official TLA+ tools v1.7.4 (TLC 2.19, revision `5a47802`), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Runs used Corretto 25.0.4.1 and Go 1.27.1. Repeat from this directory with each
configuration:

```sh
java -XX:+UseParallelGC -Xmx256m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-error-contract -config Aggregate.cfg ErrorContract.tla
```

From the repository root:

```sh
go test -race ./grpc -count=1
LOOM_DIR="$PWD" go test ./codegen/generator -run '^TestGRPCJoinedServerErrors$' -count=1
```

Keep checker traces and state outside the repository; remove them after
validation and independent review. The legacy controls must fail their named
invariants, not just fail to start the checker.

## Context termination (#597)

`ContextTermination.tla` extends the ownership analysis with canceled/deadline
leaves, a mixed cleanup join, unanimous cancellation and a native remote status.
An explicit outer contract wins. Transparent wrappers are abstracted away;
independent joins still require consensus. The client preserves a remote status
and its cause instead of replacing it with a synthetic fault. Local context state
is not an input to remote error interpretation.

Use the same pinned TLC command with `ContextTermination.tla` and these configs:

- `ContextLegacy.cfg`: `ServerContract` counterexample (exit 12).
- `ContextClientLegacy.cfg`: `ClientCause` counterexample (exit 12).
- `ContextChecked.cfg`: both invariants pass (30 distinct states, exit 0).

`classifyError` recognizes exact context sentinels after explicit owners and
before recursive unwrapping; encoding emits native termination status without
synthetic service details. Generated unary/stream-opening clients share a single
detail-dispatch path, preserving unrecognized remote errors just as stream
receivers do. `TestContextErrorContract` checks wrappers, mixed/reversed/unanimous
joins and explicit owners. `TestGRPCClientErrorCause` also checks compiled unary,
stream-opening and receive paths with active/ended local contexts, ordinary
wrappers and absent/unrecognized status details. Existing generic and declared
error checks remain; `TestGRPCJoinedServerErrors` also sends context termination
through real generated unary and streaming servers.

The bounded model does not prove protobuf detail retention, arbitrary error
traversal, real RPC delivery or context races. Those remain runtime obligations.
No retry traits are inferred from a termination status.
