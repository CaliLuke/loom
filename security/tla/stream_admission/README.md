# Stream admission readiness model

This is design evidence for #743, not a proof of the Go implementation.
`StreamAdmission.tla` models resource admission, initial frame completion,
application boundaries and original cancellation. `AuthenticationBranches.tla`
models the existing ordered AND/OR callback/context contract separately.

Recommended policy: when all credentials are available at stream open, execute
shared authentication before waiting for the first frame, then retain the normal
complete-payload endpoint checks. Mixed metadata/message alternatives wait for
complete input and retain declared order. Callback repetition is deliberate.

## Checks

Run from this directory with Corretto 25.0.4.1 and the pinned TLC jar:

```sh
java -XX:+UseParallelGC -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-admission-states -config admission-checked.cfg StreamAdmission.tla
```

Use a distinct metadir per run. The jar is release 1.7.4 (TLC reports 2.19), SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.

| Configuration | Result | Generated / distinct states | Depth |
| --- | --- | --- | --- |
| admission-checked | Pass, all five real invariants | 504 / 422 | 9 |
| checked (endpoint-object alternative) | Pass, including exactly once | 480 / 404 | 8 |
| legacy | Expected failure: PromptRejection at initial wait | Initial counterexample | 1 |
| eager-available | Expected failure: DeclaredWinner | 57 / 57 | 2 |
| lost-context | Expected failure: CancelledInvocation | 339 / 267 | 7 |
| no-recheck | Expected failure: MiddlewareRechecked | 406 / 348 | 7 |
| exactly-once-hypothesis | Expected failure of a **withdrawn hypothesis** | 315 / 286 | 5 |
| branches-checked (AuthenticationBranches) | Pass, three invariants | 58 / 58 | 5 |
| branches-leak (AuthenticationBranches) | Expected failure: WinningContext | 50 / 50 | 4 |

The exactly-once result does not reject the recommended design. The public
protected-boundary contract permits repeated checks; no callback GoDoc promises
exactly-once invocation. It records why an initially considered endpoint-object
migration is unnecessary for this repair.

## Implementation correspondence

- `grpc.ServeStream` / generated `writeGRPCStreamInputDecode`: initial wait and
  metadata authentication admission via `StreamServerSpec.Admit`.
- `grpc.AdmitStream` and generated metadata helper: detached metadata snapshot,
  physical parsing/validation, partial payload and complete envelope validation.
- `buildEndpointAuth`: ordered OR alternatives and AND scheme callbacks.
- `security.RunCheck` / `security.Protect`: original cancellation and middleware
  continuation checks.
- Ordinary endpoint / authorization check: deliberate complete-request recheck.
- `grpc/codegen/stream_admission_test.go`: generated bufconn no-frame rejection,
  callback context replacement, full payload, revocation and middleware mutation.
- `grpc/codegen/stream_admission_alternatives_test.go`: ordered AND/OR, anonymous,
  mixed availability and failed-branch context reset.
- `grpc/admission_test.go` and `security/check_test.go`: snapshot, observation,
  original cancellation and error precedence.
- The receive-first regression reproduces at `651372ce` and integration parent
  `433d2e74`, before implementation; both first-frame controls pass.

The model abstracts authentication results as fixed booleans. It does not prove
nil-context handling, metadata snapshots across context replacement, real parser
behavior, changing authorizer state (revocation), arbitrary callback termination,
Go middleware factories, runtime races or generator compilation. Those remain
explicit test obligations in the readiness document. Original cancellation may
occur at every modeled active phase. A context returned by a callback is modeled
as live, so the lost-context negative control represents a callback replacing
its context with an uncancelled one.

No checker traces, state directories or raw logs belong in the eventual commit.
