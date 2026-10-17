# Failed-stop eviction test ordering

`StopEvictionTest.tla` models the callback gate in
`TestEvictionStopsJobRestoredByFailedStop`, rather than changing the worker's
runtime contract. The failed main CI run
[38646089002](https://github.com/CaliLuke/loom/actions/runs/38646089002) shows
the event loop holding the gated callback, the test blocked in `Worker.stop`,
and the direct `stopJob` caller waiting for the transition lock.

## Decision and correspondence

- `Publish`: `stopJob` calls `issueStop`, which publishes a stream event before
  processing the returned operation directly. Either the direct caller or
  `handleEvents` can reach `handler.Stop` first (`Enter`).
- `Join`: `Worker.stop` joins the event loop. In the original test this call
  preceded `close(handler.release)`, so a runtime-owned callback prevented the
  test from ever releasing its own gate. `current.cfg` reproduces that cycle
  with a `NoJoinCycle` violation (5 generated / 4 distinct states, exit 12).
- `Quiesce`: the corrected test joins the runtime before issuing the gated
  stop. Only the direct caller can now enter the callback. The later join is
  vacuous because no runtime goroutine remains.
- `Release` / `FailStop` / `Evict`: eviction waits for the failed callback;
  the job remains tracked on failure and eviction retries Stop and forgets it.
  The Go test checks the handler error, retained job while blocked, eviction
  completion, empty final jobs, and exactly two Stop calls.

`quiesced.cfg` passes all three invariants (8 generated / 8 distinct states,
exit 0). This bounded model assumes finite runtime shutdown before publication,
successful eviction's second callback, and no other ownership transition. It
does not prove Redis delivery, lease timing, general worker shutdown liveness,
or mutex implementation correctness. The runtime behavior and production locks
are unchanged. Repeated and race-enabled Go tests check the actual test harness.

## Run

Use TLC 2.19 from `tlaplus/tlaplus` release `v1.7.4` (`tla2tools.jar`, SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`), Java 25:

```sh
cd pulse/pool/tla/eviction_test
java -XX:+UseParallelGC -cp /path/to/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/loom-eviction-current-states \
  -config current.cfg StopEvictionTest.tla
java -XX:+UseParallelGC -cp /path/to/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/loom-eviction-quiesced-states \
  -config quiesced.cfg StopEvictionTest.tla
```

`-deadlock` allows the normal terminal state; `NoJoinCycle` identifies the
invalid circular wait. Keep checker traces and state directories outside the
repository and remove task-owned output after review.
