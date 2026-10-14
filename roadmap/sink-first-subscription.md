# Sink first-subscriber activation — #901

The independently reproduced defect is loss before the caller can register any
subscriber: both XREADGROUP and idle reclaim can advance remote delivery then
fan out to no channels. Acknowledged entries remain pending; NOACK entries have
no ordinary retry. Existing constructor membership is not local delivery readiness.

1. Preserve current read/claim negative probes and independent receipts. Add
   candidate-positive tests against the public delivery contract; do not wait for
   an empty fanout in a corrected test. Cover acknowledged and NOACK originals,
   initial pending claims, concurrent first subscriptions, and Close before the
   first subscription. Reuse existing Redis hooks/manual idle triggers.
2. Keep membership creation and keepalive eager. Register a subscriber and activate
   both read and idle-claim workers once inside the existing lifecycle mutex.
   A startup token/flag belongs to this shared activation transition. Account for
   both workers before launch so Close cannot finish before an admitted worker.
   Reject late Subscribe using existing sealed admission. Do not add a second
   lifecycle coordinator or per-producer ad hoc checks.
3. Align the existing subscriber admission model and the new bounded activation
   model with implementation. Eager startup fails before first subscription;
   deferring only reads still loses claimed entries. Atomic activation passes
   under eventual worker-exit assumptions. Models do not prove socket interruption,
   unbounded fairness, all-unsubscribed replay, or actual CI timeout causation.
4. Update existing tests that intentionally exercise reads or claims without
   subscribing, including #768/#800 probes, so lazy activation cannot hide the
   behavior they measure. Preserve active shutdown regression evidence separately;
   no patched go-redis dependency or #800 claim belongs in this change.
5. Run focused race tests, affected streaming tests, and a bounded native Redis
   correspondence check using a disposable loopback server. Use the pinned latest
   stable Redis and ordinary released dependency. No generated output is affected.
6. Document that consumption begins with first successful Subscribe, with no new
   promise after all subscribers unsubscribe. Keep proof sources/configuration,
   run instructions, model limitations and implementation/test correspondence next
   to pulse/streaming; remove temporary checker output only after review.
7. Fresh independent final review, one atomic issue commit, push and closure.

Candidate configuration currently passes 872 generated / 313 distinct states.
Current and partial configurations fail NoEarlyAdvance for read and claim
respectively, both in five generated states. See the independent assessment at
/tmp/loom-sink-startup-claim-independent-assessment.json. The prior read-only
assessment is /tmp/loom-sink-startup-independent-assessment.json.


## Implemented owner and acceptance

The first admitted `Sink.Subscribe` now registers its channel, consumes
`deliveryStarted`, accounts for two workers and launches read plus claim while
holding the existing lifecycle mutex. `newSink` starts only keepalive. The two
superseded eager launch paths are removed; no loop-specific activation check or
second lifecycle coordinator is introduced. Close still seals admission under
the same mutex before waiting. This does not repair its independent I/O bounds
or concurrent-Close joining behavior.

- [x] On exact parent `481bdf788`, unchanged negative probes reproduce both
  original-event losses and the foreign pending-entry transfer before Subscribe
  (race 0.958s). The Go overlay replaces only `sink.go` with its exact parent blob;
  the new activation-only tests are omitted from that parent probe build.
- [x] Candidate activation, active/unused subscriber shutdown and manual rotation
  controls pass under race (2.373s); full streaming package race passes (7.409s).
- [x] Twenty focused concurrent activation/Close repetitions pass under race
  (4.026s).
- [x] Disposable native Redis 8.10.2 activation/admission and pending recovery
  controls pass (2.437s). No patched Redis dependency is used.
- [x] Both negative model configurations fail `NoEarlyAdvance` in five states;
  the candidate passes 872 generated / 313 distinct states and closing progress
  under its documented worker-exit assumption.
- [x] Existing active #683 shutdown tests remain active; unused and started Sink
  are now separate controls. The manual rotation fixture directly registers its
  channel beside its manually launched read worker, without pretending that its
  partial Sink owns automatic keepalive/claim startup.
- [x] Original #768/#800 reproduction inputs remain unchanged. Copied variants
  explicitly subscribe: a matching delivery barrier still exposes five filtered
  entries left pending (#768, native race 0.924s), and Redis CLIENT LIST proves an
  active BLOCK 0 read before the unchanged Close/concurrent-Close failures (#800,
  native race 0.536s). The #800 implementation must retain this activation step;
  lazy startup is not evidence of repaired active shutdown.
- [x] Scoped streaming lint passes with zero issues; formatting and diff checks
  pass. No code-generation, dependency or protocol serialization change.
- [ ] Fresh independent final review by an agent who did not contribute this
  design/implementation, atomic delivery, then inactive-artifact cleanup.

Exact original and adapted probes are preserved under `/tmp/loom-901-inputs`;
`/tmp/loom-901-parent-overlay.json` and `/tmp/loom-901-residual-overlay.json`
record source selection. The first combined residual test selector ran only
#768; the separate exact #800 selector supersedes it for shutdown evidence.
