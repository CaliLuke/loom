# Guard fixture revision observation (#906)

The pool fixture waits for an exact replica value after publishing a stale guard.
The real cleanup owner can remove that guard before the wait observes it. CI's
intent-retry failure and a deterministic cleanup-before-reply probe expose this
fixture race. The runtime already owns the correct revision acknowledgment.

1. [x] Reproduce the permitted interleaving and inspect all helper callers.
2. [x] Obtain independent readiness review; retain production semantics and every
   dispatch, pending-entry, incarnation and cutoff assertion.
3. [x] Replace the shared fixture's value polling with existing `SetAndWait` and
   a bounded context. Preserve the exact five-second waiting budget.
4. [x] Keep a deterministic regression that observes the write and its cleanup
   before returning the Set reply. Run the original intent case and all direct
   guard consumers under race, plus scoped lint.
5. [ ] Independent exact-final review, one atomic ticket commit, push and close.

This is test synchronization, not a new pool protocol or formal-model change.
`rmap.SetAndWait` acknowledges a committed revision even after supersession; the
old equality barrier never guaranteed that the value persisted after it returned.
Tests requiring an extant guard must own competing cleanup separately. No timeout
increase, production scheduling change or weakened operation assertion is needed.

The deterministic parent regression fails at the original equality wait in
5.773s, independently reproduced in 5.494s. Twenty ordinary repeats pass, so the
forced permitted interleaving is the useful regression. Validation covers the
original intent retry, every guard helper consumer, native dispatch/ack status
assertions and the cleanup-before-reply case under race; scoped lint is clean.
