# Subscriber lifecycle admission — #683

The Reader and Sink lifecycle mutex is the publication boundary. The shared
`registerSubscriber` helper allocates a channel while that mutex is held, closes
it immediately if `closing || closed`, and otherwise registers it. Only successful
Reader admission can consume `startOnce` and start the read worker. Close seals
admission under the same lock before it drains existing subscribers.

`SubscriberAdmission.tla` models two subscriptions, shutdown admission, final
drain and unsubscribe. It distinguishes a rejected channel from a registered
channel and records whether a late subscription started the first worker.

- `current.cfg`: expected `NoLateWorker` counterexample (exit12), Close before
  first Subscribe followed by worker startup after admission should be sealed.
- `current-channel.cfg`: expected `LateChannelsClosed` counterexample (exit12),
  including a channel returned open while shutdown is in progress.
- `candidate.cfg`: passes all invariants and closing progress,60 generated states,
  34 distinct. No late worker; every late returned channel is already closed;
  completed owners retain no subscriber; unsubscribe remains safe.

Run with TLC2.19 / Java25 and cached tla2tools-1.7.4.jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-683-model/candidate \
  -config candidate.cfg SubscriberAdmission.tla
```

Run the two negative configurations likewise. Keep checker outputs under `/tmp`.
The model abstracts the lifecycle critical section as an atomic action and assumes
existing workers eventually drain; it does not prove socket cancellation, prompt
Close, concurrent Close joins or subscriber send scheduling. Those #800 concerns
remain separate. No new Redis timeout or dependency change belongs to this repair.

`subscriber_admission_test.go` maps the model to both public Subscribe methods:
closed and closing owners, unused and already-started Reader, 24 subscriptions
racing Close, and normal delivery/unsubscribe followed by resubscription. An
additional unused-reader test checks that late Subscribe cannot consume the
startup token even if an incorrectly started worker exits before inspection.
Closing tests hold an admitted read using a gate and always release/join it;
they do not rely on unbounded Redis cancellation or leave parent goroutines behind.

At parent `6486dfb45528a6aa8ff1bfe02e36f0472f58a9fc`, direct race tests fail for
late open channels, retained subscribers and worker startup. Corrected source
passes the whole streaming package with race in6.699s, targeted admission tests
against Redis8.10.2 in1.894s, and20 focused concurrency repetitions in2.862s.
The later startup-token case was checked separately on parent/candidate and real
Redis; no production change invalidated the earlier package evidence.

```sh
go test -race ./pulse/streaming -count=1
go test -race ./pulse/streaming -run '^TestSubscriberAdmissionConcurrentClose$' -count=20
LOOM_PULSE_REDIS_ADDR=127.0.0.1:32768 go test -race ./pulse/streaming \
  -run '^TestSubscriberAdmission' -count=1
```

The Redis helper flushes only its streaming test database. Use a disposable
loopback Redis and do not overlap another task's probes. Tests use released
v9.23.0; the #800 local patched dependency is not used.
