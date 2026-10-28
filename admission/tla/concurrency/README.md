# Unary admission and resizing

`Concurrency.tla` explores permit lifetime and live capacity changes before the
Go implementation. This is a new capability, not a reproduction of a previously
supported limiter defect. One mutex owns admission, capacity updates, completion
and snapshots in `../../concurrency.go`.

The checked model allows arbitrary ordering of admission, rejection, cancellation,
completion and capacity updates. Nonpositive capacity is represented by zero and
leaves state intact. Capacity reductions can leave more running calls than the
new limit; the safety requirement applies to each new admission, not retroactively
to running calls. Repeated updates never discard permits.

Assumptions and scope:

- Each of three request identities invokes once. Finish represents success,
  error or panic unwinding; implementation tests cover those concrete paths.
- Admission and resize are atomic actions, corresponding to mutex ownership.
  Cancellation can precede or follow admission. It cannot terminate an invocation
  by itself. There is no fairness or completion-time guarantee.
- `owned = running` checks permit retention, and `Accounting` checks disjoint
  active/completed/rejected identities. Counts are abstracted as sets; this is not
  a proof of Go defer semantics, integer overflow or exactly-once execution of
  arbitrary application middleware.
- The model excludes streams, process failures, transport/authentication behavior,
  distributed limits, queues and background work after endpoint completion.

Using the repository's pinned TLC distribution (`tla2tools-1.7.4.jar`, reporting
TLC2 2.19, revision 5a47802) and Java 25.0.5, from this directory:

```sh
java -XX:+UseParallelGC -cp "$HOME/.local/share/tlaplus/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-admission-model \
  -config checked.cfg Concurrency.tla
```

Run `cancel.cfg` and `resize.cfg` explicitly as negative controls. Neither is a
legacy implementation: they represent rejected candidate designs. The first
releases a permit on cancellation, and the second forgets permits when resizing.
Both violate `PermitLifetime` after admission followed by the named action.
Checked exploration passes 1,425 distinct states (10,072 generated). Cancellation
fails after 10 distinct states; resize fails after 16. The model informed the
choice to release only on endpoint exit and update capacity in the same owner.

`../../concurrency_test.go` checks configuration, context identity, simultaneous
admission, each exit path, cancellation, resizing below usage and concurrent
updates/completions. `../../../codegen/generator/admission_integration_test.go`
(relative to the repository root: `codegen/generator/`) generates the runnable
example and checks HTTP/gRPC mappings and protected validation/access ordering.
The model does not establish those transport properties. Run the direct Go tests
with `go test -race ./admission` and the generated test with local Loom source
selected while developing an unpushed change.

All formal commands are opt-in design tools. They are not CI or release gates.
Remove checker output and traces after review; keep these sources and findings.
