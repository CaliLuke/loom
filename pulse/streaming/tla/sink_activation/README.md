# Sink first-subscription activation — #901

`Sink.Subscribe` owns registration and the one-time activation of both delivery
workers under the lifecycle mutex. `newSink` creates membership and starts only
keepalive. The wait group accounts for both delivery workers before launch;
`Close` seals admission under that same mutex before waiting.

The model compares eager delivery (`current.cfg`, `NoEarlyAdvance` fails for a
read), deferring only reads (`partial.cfg`, fails for a claim), and atomic
registration/activation (`candidate.cfg`, passes: 872 generated / 313 distinct
states). Each negative reaches the invariant failure in five generated states.
The `starts` type invariant permits at most one launch per delivery worker.

Run TLC 2.19 with the existing pinned tla2tools-1.7.4.jar (SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-901-model/candidate \
  -config candidate.cfg SinkActivation.tla
```

Run `current.cfg` and `partial.cfg` likewise; both must exit 12 with invariant
counterexamples. Keep checker state outside the repository.

The finite model has two subscribers and one modeled delivery per worker. It
abstracts the lifecycle critical section and assumes workers eventually exit
and drain. It does not model Redis pending-entry mechanics, blocking socket
cancellation, concurrent Close joining, post-Close remote mutations or eventual
replay after every subscriber unsubscribes. Activation remains true after
Unsubscribe; no new all-unsubscribed guarantee or #800 repair is implied.

`sink_activation_test.go` checks acknowledged and NOACK original delivery,
foreign pending-entry claim after subscription, and concurrent first admission.
`subscriber_admission_test.go` retains unused/started shutdown, sealed late
admission and active read/Close races. The original negative probes observe an
actual empty fanout before Subscribe; the positive tests do not wait for that
forbidden transition. Native Redis checks cover the pending-entry behavior that
the model abstracts. See also the adjacent subscriber_admission model.


Candidate validation: full streaming race 7.409s; focused native Redis 8.10.2
activation/admission/pending-recovery race 2.437s; scoped streaming lint passes.
Copied #768/#800 probes explicitly subscribe and still reproduce their separate
filtered-entry and active-close defects. Their original historical inputs remain
unchanged. The implementation does not claim those repairs or aggregate CI green.
