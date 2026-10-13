# Pending replicated-map notification precedence

This bounded model isolates #771's single-subscriber channel. Kinds 1/2/3 mean
Change/Delete/Reset; zero means empty. At most three live or reconnect aggregate
notifications arrive. `owed` tracks the strongest unapplied consumer obligation.
The receiver can run between every publisher step; the map lock excludes another
publisher or Unsubscribe/Close until publication finishes. Close may discard the
subscription's obligation, while an already-buffered event remains readable.

`current` drops a new kind behind a full slot. `replace` drains and blindly
replaces, losing an older stronger kind. `owned` drains and combines with the
existing maximum-priority rule. The invariant checks completed publications;
a consumer can legitimately take an older value while publication is in flight,
but the new obligation must subsequently remain available. Fair publication and
receive steps check eventual delivery unless ownership closes. This is not a
proof of Redis pubsub delivery, scheduler task issuance or #800 resource shutdown.

Run each configuration using the shared pinned TLC tool and external state:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-771-model-owned -config owned.cfg NotificationPrecedence.tla
```

Use `current` or `replace` instead of `owned` for expected counterexamples.
The implementation seams are `Map.notifySubscribers`, `strongerEventKind`,
`Subscribe`/`Unsubscribe`, and the run goroutine's closure ownership. The same
notifier receives live changes and reconnect aggregation. Direct priority and
consumed-kind controls check the finite encoding; Redis-backed tests and a gated
actual scheduler subscriber check the concrete consumer.

The implementation now drains each unread slot, merges through
`strongerEventKind`, and sends the result. The final send has no default branch:
Subscribe creates capacity-one channels and the map lock excludes every other
publisher, so after the nonblocking take the slot is necessarily available.
A receiver can only empty it. The model's owned Publish therefore always takes
its empty-buffer branch; current-mode Publish retains the defective drop branch
for the counterexample.

TLC 1.7.4 results: current drop fails `PendingRetained` after 26 generated/24
distinct states; blind replacement fails after 59/42; owned priority merging
checks all invariants and eventual delivery over 170/98. The model represents
one subscriber because each has independent pending state under the same sole
publisher lock. `TestNotificationSubscribersRemainIndependent` additionally checks that merging
one channel does not leak its kind into the next. Close represents both
serialized unsubscribe and final run-goroutine ownership release; Go tests check
their distinct concrete paths. Buffered receives after closure remain permitted.
