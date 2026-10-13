# Per-request logging scope (#746)

The root can be empty, buffered or previously flushed. Two requests start in
any order, each logs and flushes once. Request IDs are enabled or disabled.
Current behavior reuses root storage without IDs; always calling With isolates
the pointer but copies root buffered entries and disabled-buffer state. Fresh
request scope inherits configuration/fields, with empty entries and a new flush
state. The invariants require independent ownership, no foreign entries and
buffering independent of an earlier scope's flush.

This bounded model assumes default buffering policy and abstracts entries by
owner tokens. Debug and DisableBuffering callbacks intentionally opt out and
remain direct runtime obligations. It does not model field derivation within a
request (#745), application retention after request return, memory size, callback
reentrancy, writers, cancellation or logger configuration races (#744). Request
scope does not forcibly flush or cancel application contexts on handler return.

Run from this directory with TLC1.7.4, JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
Corretto Java25.0.4.1. Use separate temporary JVM and state directories per run:

```sh
mkdir -p /tmp/loom-746-java-fresh
java -Djava.io.tmpdir=/tmp/loom-746-java-fresh -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 -metadir /tmp/loom-746-state-fresh -config fresh.cfg RequestScope.tla
```

Current must fail ScopeOwnership; clone must fail NoForeignEntries or
IndependentBuffering. Fresh must pass all invariants. Expected invariant failures
must be semantic counterexamples, never checker syntax/configuration errors.
Implementation correspondence is the shared private request-context constructor,
called by HTTP and both gRPC server interceptors with optional IDs supplied as
initial fields, so request buffering policy sees those fields and runs once.
`TestRequestLoggerScope` covers three transports, both ID/logging options and
three root states, retaining fields/request values and explicit request flushes.

Observed: current violates ScopeOwnership at7/7states, clone violates
IndependentBuffering at7/7, fresh passes150generated/96distinct. Runtime tests
also demonstrate cloned root buffered entries crossing into ID-enabled requests;
that alternative can violate NoForeignEntries, beyond TLC's first reported trace.
Debug and request-value-based buffering policy controls preserve application
configuration, canceled status, deadlines and values across all three transports.
