# Activity registration and workflow-scope binding

`ActivityBinding.tla` checks the adapter boundary extracted into
`execution/selvedge`. Registration publishes a local definition only after native
registration succeeds, and futures keep the workflow scope that scheduled them.
The registry lock spans both registration steps. A scope is an immutable value;
there is no registry-wide "current workflow" context.

Run with TLA+ tools v1.7.4 (TLC 2.19), Java 25, from this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/activity-binding-checked \
  -config checked.cfg ActivityBinding.tla
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/activity-binding-rebound \
  -config rebound.cfg ActivityBinding.tla
```

The checked configuration passed: 53 distinct states, 321 generated states.
The rebound configuration must fail `FutureScopeIsStable`: it models an
alternative that selects the caller's later scope during `Get`, moving a future
from its parent scope to a detached scope. TLC found that four-state trace.

Correspondence tests are `TestNativeConflictDoesNotPublishLocalDefinition` and
`TestActivityScopesStayIndependent`. Agent conformance and saved workflow histories check the shared `execution`
contracts with agent-owned serializers, error conversion and defaults. The
consuming application's metadata tests exercise this native Selvedge binding.

Bounds: two names, two scopes, two futures. Native registration is assumed to
reject conflicts without replacing the installed handler. Selvedge owns command
ordering, retries, persistence, transactions, and cancellation implementation;
this model does not prove those backend properties. Native transaction admission
is retained and tested in the consuming application rather than reimplemented
in this adapter. The model is a design check, not a claim that the extracted
implementation previously had a scope-rebinding defect.
