# Release fault admission (#947)

This bounded model concerns the test fault injector, not the production release
protocol. Direct and asynchronous consumers share one job lock; another worker
can release a different job concurrently. Current code admits any caller while
its fired flag is false, then marks that flag after the gate. An asynchronous
consumer can own the injected error, and two unrelated release calls can enter
the same gate before the flag changes.

The owned design scopes the hook to its explicit direct-operation context and
claims the fault atomically before signaling or waiting at the gate. Its direct
lease fixture also owns delivery by closing the pool sink before the test job
starts. Context identity alone does not prevent an asynchronous consumer from
finishing the job first. The separate real-public-path control retains that
valid asynchronous-first outcome.

Run with `TLA2TOOLS_JAR` pointing to the installed, pinned tla2tools v1.7.4
jar (TLC 2.19). Set the variable before these commands:

```sh
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/release-current-caller -config current_caller.cfg ReleaseFaultOwner.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/release-current-gate -config current_gate.cfg ReleaseFaultOwner.tla
java -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/release-owned -config owned.cfg ReleaseFaultOwner.tla
```

Current caller admission fails FaultBelongsToDirect after the async consumer
enters and claims the hook (9 generated / 7 distinct states). Current check/store
admission fails GateHasOneOwner when two calls enter before either sets fired
(30 / 18). Owned admission passes both safety checks and eventual direct error
(20 / 12), assuming weak fairness and that the test releases its gate.

`Enter` corresponds to stopOperation's job lock; `Admit` to the context check and
fired CompareAndSwap in releaseReplyHook; `Fault` to unblocking the gate and
returning the injected error. The controlled publication-reply regression checks
that real Redis delivery can precede the direct caller's receipt. Hook tests
cover context mismatch, simultaneous attempts, and after-write NOSCRIPT fallback.
Lease renewal remains checked against actual advancing Redis timestamps and
cleanup refusal in TestSlowReleaseKeepsLease. Network timing, Lua correctness,
application callbacks and arbitrary crashes remain outside this small model.

Verified jar SHA256:
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
