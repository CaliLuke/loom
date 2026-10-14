# Worker publication and local admission

This is the bounded continuation of the earlier `WorkerPublication` readiness
model in `pool-construction-ownership/loom`. That model distinguished independent
membership/heartbeat replicas but required every queued worker event to retain a
runtime. Its retained-drain candidate omitted the actual pool-event recovery
path. This version models that path instead of requiring a new draining runtime.

`s.prepared` includes an allocated, **unsubscribed** Reader. Only `Activate`
starts worker loops and transfers ownership into the node. `BeginClose` excludes
admission; owned `FinishClose` waits for the constructor. `AbortConfirmed`
withdraws the private stream and both publications. `AbortUncertain` explores
every subset of committed deletions, including none, without starting a runtime
or renewing the initial heartbeat. Local resources are released in every case.

`Route` represents XADD NOMKSTREAM: stale independent replicas can route only
while the private stream exists. Every route retains the original unacknowledged
pool event. Only handling it acknowledges it. `Recover` represents another healthy
worker processing the reclaimed original after the failed worker's lease expires;
it is not a handler call on the failed attempt.

## Checks

| Config | Expected result | Generated / distinct states |
| --- | --- | --- |
| early | PublicationHasPreparation fails | 6 / 6 |
| publish_last | NoWorkerAfterClose fails | 532 / 202 |
| withdraw_only | ConfirmedAbortRemovesPrivateStream fails | 33 / 21 |
| ack_abort | OriginalEventPreserved fails | 10 / 10 |
| owned | All six safety invariants pass | 3158 / 940 |

The first two reproduce current publication/Close ordering and the insufficient
publish-last alternative. The latter two reject membership-only withdrawal and
acknowledging discarded queued events. These are design alternatives, not claims
that the current routing implementation acknowledges discarded events.

Run with TLA+ 1.7.4 (`tla2tools` SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)
and Java 25 from this directory:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-worker-publication-owned -config owned.cfg WorkerPublication.tla
```

Repeat with each negative configuration, expecting exit 12 and the named invariant
counterexample. Exit 0 is expected only for owned. Keep checker output outside the
repository. `/tmp/loom-704-model-results.json` records the readiness run.

## Correspondence and limitations

Implementation: `Node.AddWorker` registers `workerAdmissions` under the lifecycle
lock; `prepareWorker` owns the unstarted resource state. The successful transfer
calls `Worker.start` under that lock. `Node.close` fences and joins attempts before
taking its worker snapshot. `abortAdmission` releases local resources and attempts
stream/membership/heartbeat withdrawal, returning all failures to the caller.
Individual Redis deletions are not atomic: uncertain cleanup conservatively
allows every surviving subset, and routing still requires the stream to exist.

- `TestConstructionWorkerIsNotPublishedBeforeReady` checks the current early
  membership/heartbeat publication before private stream initialization.
- `TestConstructionWorkerLostHeartbeatReply` injects failure **after** Redis
  commits, including cancellation of the caller. Successful withdrawal must be
  checked against authoritative maps, not inferred from another replica.
- `TestConstructionWorkerCannotOutliveNodeClose` joins both real operations and
  checks that no new live local worker escapes Close.
- `TestWorkerAdmissionRecovery` checks actual XADD NOMKSTREAM, PEL retention,
  XAUTOCLAIM and routing to a healthy worker after private-stream deletion.
- `TestWorkerAdmissionUnstartedReader` checks abort release even with BLOCK 0;
  no read was started. Existing active-worker Close remains a separate lifetime.
- `TestWorkerAdmissionPrivateFailures` covers failed writes, canceled replica
  waits and a lease that expires before local transfer, preserving a healthy peer.
- `TestWorkerAdmissionCleanupErrors` verifies all private cleanup attempts and
  returned errors. `TestWorkerAdmissionFailedCleanupDoesNotRenew` checks the
  surviving-publication branch without handlers or periodic heartbeat renewal.
- `TestWorkerAdmissionQueuedBeforeFailure` connects committed-publication failure
  directly to private queued-copy withdrawal and healthy original-event recovery.

This is a safety model, not an availability or network-deadline proof. Recovery
assumes a healthy peer and delivery within existing stream retention/event TTL;
it does not extend DispatchJob's promise beyond its existing timeout. It abstracts
lease expiry and Redis execution order, does not prove clock-skew bounds, Redis
durability, arbitrary delayed commands after connection failure, cancellation of
blocked handlers, or recovery through an indefinite outage. A failed cleanup may
leave stale publication until the initial lease expires; no heartbeat renewal is
allowed. Tests check implementation correspondence, including real Redis behavior.

## Node construction (separate #687 owner)

See [NodeConstruction.md](NodeConstruction.md) for the current node-only acquisition
model, explicit #800 finite-close assumption, rejected rollback alternatives and
run commands. WorkerPublication and its delivered #704 evidence above are unchanged.
