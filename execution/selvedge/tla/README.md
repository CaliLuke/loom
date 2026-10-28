# Activity registration and workflow-scope binding

`ActivityBinding.tla` checks the adapter boundary extracted into
`execution/selvedge`. Registration publishes a local definition only after native
registration succeeds, and futures keep the workflow scope that scheduled them.
The `owned` set records successful adapter registrations separately from all
native names. Every published definition must belong to that set.
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
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/activity-binding-conflict \
  -config conflict.cfg ActivityBinding.tla
```

The checked configuration passed: 53 distinct states, 321 generated states.
The rebound configuration must fail `FutureScopeIsStable`: it models an
alternative that selects the caller's later scope during `Get`, moving a future
from its parent scope to a detached scope. TLC found that four-state trace.
The conflict configuration must fail `PublishedDefinitionsExist`: it publishes
a local definition after an independent native registration claims the same
name. TLC found that three-state trace, with no adapter-owned native definition.
Both negative configurations exit with status 12; the checked one exits with 0.

Before tracking ownership, the invariant checked only that every published name
existed in the native registry. Removing the native-conflict guard still passed
that weaker invariant (53 distinct states, 409 generated states). The ownership
invariant and conflict configuration close that proof gap; runtime registration
already rejected this case.

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

## Isolating retry-policy tests from attempt deadlines

`PolicyTest.tla` checks the assumption behind exact invocation counts in the
adapter's error-policy tests. A permanent handler result stops retries only if
it wins the attempt outcome. An independent attempt deadline can expire first
and legitimately cause another invocation. Delayed scheduling is nondeterministic
in the model; it does not assume that a short handler runs promptly.

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/policy-deadline \
  -config policy-deadline.cfg PolicyTest.tla
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/policy-isolated \
  -config policy-isolated.cfg PolicyTest.tla
```

The former one-second attempt deadline admits a counterexample to `SingleAttempt`:
claim, expire, claim again, then return the permanent error. The isolated policy
configuration has no attempt deadline and satisfies the invariant. A native
adapter probe holding the first handler for 1.5 seconds reproduced two invocations
with the old deadline, an expired first context and a final permanent error; the
same probe without the deadline produced one invocation and no context error.

`TestNativePermanentErrorPolicy` and `TestNativePolicyAndTypedResults` therefore
test error policy without an unrelated attempt deadline. Their enclosing test
context still bounds completion. `TestActivityTimeoutPolicy` separately tests
native default and portable override deadlines by awaiting context cancellation,
checking the typed timeout and the recorded policy; it makes no prompt-scheduling
or exact invocation-count assertion. The options-merge test also omits an unrelated
deadline. No runtime retry behavior changes.

Bounds and assumptions: two attempts, one possible deadline expiry, permanent
result commitment ends the activity. Native persistence, claim renewal, crashes,
and workflow cancellation are outside this model; it isolates the test's invalid
assumption rather than proving exactly-once backend execution. The controlled
probe establishes a causal reproduction, not proof of the scheduler history in
the original CI run. These model checks remain explicit, opt-in design tools.
