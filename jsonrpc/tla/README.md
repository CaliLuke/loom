# TLA+ model of the JSON-RPC WebSocket client connection

`WebSocketClientDemux.tla` models how a generated JSON-RPC WebSocket client
routes responses to the streams that share its connection (issue #399). It
covers the design on `main` at commit `c77c90e0` and the demultiplexer in
`jsonrpc/websocket_client.go` that replaced it.

## Files

| File | Purpose |
| --- | --- |
| `WebSocketClientDemux.tla` | The model: state, actions, invariants, liveness, designs. |
| `cfg/asis.cfg` | Main before #399: the gorilla one-reader and one-writer rules. |
| `cfg/asis_routing.cfg` | Main before #399: can a stream get another stream's response? |
| `cfg/asis_lost.cfg` | Main before #399: can a response be dropped while its stream waits? |
| `cfg/one_reader.cfg` | One reader, but ids still numbered per stream. |
| `cfg/unguarded.cfg` | One reader and connection ids, registration not guarded. |
| `cfg/demux_split.cfg` | The first `Release`: drops the last reference, unlocks, then marks and closes the connection. |
| `cfg/demux.cfg` | The implemented design, all faults, every safety invariant. |
| `cfg/demux_live.cfg` | The implemented design, liveness of a waiting `Recv`. |

## Running TLC

TLC ships in `tla2tools.jar` (tested with TLC 2.19 on Java 25).

```sh
curl -L -o /tmp/tla2tools.jar \
  https://github.com/tlaplus/tlaplus/releases/latest/download/tla2tools.jar
cd jsonrpc/tla
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/tlc-states \
  -config cfg/asis.cfg WebSocketClientDemux.tla
```

- `-deadlock` is required. A state with no enabled action is a normal end
  state (every request sent and answered), not a bug.
- `-metadir` keeps the TLC `states/` directory out of the source tree.
- `-workers 1` gives a strict breadth-first search, so the first
  counterexample is a shortest one. The violating configurations finish in
  about one second.
- Run `demux.cfg` and `demux_live.cfg` with `-workers auto`. On 10 cores,
  `demux.cfg` takes 3 to 8 minutes and `demux_live.cfg`, which also checks
  liveness, 30 to 50 minutes.

## Abstraction

Two streams (`s1`, `s2`) share one connection and each sends `MaxSends = 2`
requests. A stream registers a request under its id, queues it for `Recv`,
and then writes it. The server answers the written requests in any order and
can send one id-less notification.

| Variable | Go structure |
| --- | --- |
| `pending` | Registrations: the pending map of each stream (`asis`) or `WebSocketClientConn.calls`. |
| `queue`, `wait` | `WebSocketClientStream.queue` and the request a `Recv` waits for. |
| `wq`, `wlock` | A registered request whose write is in progress, and the `WebSocketStream` write lock. |
| `reqs`, `wire` | Requests the server holds, and messages sent to the client and not read yet. |
| `rd` | The reader goroutines: one per stream (`asis`) or one per connection. |
| `cs` | The socket. |
| `dead` | `WebSocketClientConn.err`: the connection is marked closed under its lock. |
| `refs` | `WebSocketClientConn.refs`. `Open` is `Acquire`: the ping succeeds and `err` is nil under the lock. |
| `rel` | `demux_split` only: the last `Release` dropped its reference but has not marked the connection yet (`"pending"`), or marked and closed it after a stream acquired it (`"bad"`). |
| `failed`, `st` | `WebSocketClientStream.err`, and whether the stream is open or ended. |
| `delivered` | Results held in the result channel of a request, with the stream whose request the server answered. |
| `lost` | History: responses dropped while their own stream still waited. |

Faults and environment, each behind a toggle: a `Recv` times out
(`TIMEOUT`) or is canceled and requeued (`CANCEL`), a stream is closed or its
context is canceled (`CLOSE`), the socket fails (`SOCKET_FAIL`), the client
closes the connection (`CLIENT_CLOSE`), and the server sends a notification
(`NOTIFY`). On a closed socket a write fails, or seems to succeed although no
answer will come: a read can fail while writes still succeed, for example
after the server sent a close frame.

A waiting `Recv` wakes only on the result of its request, the end of its
stream, its timer, or its context, as in the Go code. It does not watch the
connection, so the reader must fail every waiter itself.

### Designs

| `DESIGN` | Readers | Request ids | Registration |
| --- | --- | --- | --- |
| `asis` | One per stream, each with the stream context. | Per stream, from 1. | Per-stream map. Closing a stream closes the connection. |
| `one_reader` | One per connection. | Per stream, from 1. | One shared map keyed by id. |
| `unguarded` | One per connection. | Per connection. | Not refused after the reader failed the waiters. |
| `demux_split` | One per connection. | Per connection. | As `demux`, but the last `Release` marks the connection closed in a later step (`ReleaseClose`) after unlocking. |
| `demux` | One per connection. | Per connection. | Refused once the connection is marked closed, under the lock the reader takes to fail the waiters. The last `Release` marks the connection closed in the same critical section that drops the reference, then closes the socket outside the lock. |

### Properties

| Property | Meaning |
| --- | --- |
| `OneReader`, `OneWriter` | At most one goroutine reads, and one writes, the connection at a time. |
| `OwnResponses` | A stream only gets responses to its own requests. |
| `NoLostResponse` | No response is dropped while its own stream waits for it. |
| `UniqueIds` | One registration per id on the connection. |
| `NoStuckRecv` | A waiting `Recv` can always return without its timer. |
| `NoLeakAfterStop` | No registration is left after the reader stopped, except one whose write is still in progress. |
| `ClosedWhenReleased` | The connection is closed once every opened stream ended. |
| `NoCloseUnderAcquire` | The last release never closes a connection that a stream acquired after the release dropped its reference. |
| `RecvReturns` | Liveness: every waiting `Recv` eventually returns. |

## Results

| Configuration | Result | Distinct states | Depth |
| --- | --- | --- | --- |
| `asis` | `OneReader` violated: two streams open and both readers enter `ReadMessage`. | 92 | 5 |
| `asis_routing` | `OwnResponses` violated: both streams register id 1, and the reader of `s2` reads the response to `s1` and hands it to `s2`. | 380 | 10 |
| `asis_lost` | `NoLostResponse` violated: the reader of `s2` reads the response to `s1`, finds no id 1 of its own and drops it as an orphan; `s1` waits until its timeout. | 236 | 9 |
| `one_reader` | `NoStuckRecv` violated: `s2` registers id 1 and replaces the registration of `s1`, which then waits for a response that will never be routed to it. | 44 | 6 |
| `unguarded` | `NoStuckRecv` violated: the socket fails, the reader fails the waiters and stops, then a stream registers a request, its write seems to succeed, and `Recv` waits forever. | 2,586 | 9 |
| `demux_split` | `NoCloseUnderAcquire` violated: `s1` opens and closes, dropping the last reference; `s2` acquires the still-unmarked connection; the pending close then marks and closes it under `s2`, whose first send fails. | 48 | 5 |
| `demux` | No error, including `NoCloseUnderAcquire`. | 16,038,571 | 42 |
| `demux_live` | No error: `RecvReturns` holds. | 9,661,013 | 42 |

`asis` reproduces issue #399. The generated-module test
`TestJSONRPCWebSocketStreamsShareConnectionGeneratedModule` and the
integration test `TestJSONRPCWebSocketClientStreamsShareConnection` show the
same failures on the old code: data races in gorilla/websocket, a stream that
receives the result of another method, and responses reported as orphaned.

`one_reader` shows that one reader is not enough: ids must be unique on the
connection. `unguarded` shows that the check that the connection is usable
and the registration must happen under the lock that the reader holds when
it fails the waiters. `demux_split` is the first implementation of
`Release`, found in review: it must mark the connection closed in the same
critical section that drops the last reference.
`TestWebSocketClientConnAcquireAfterLastRelease` fails on that version.
`demux` is the implemented design.

## Client closure and redial (#400)

`ClientClose.tla` models the generated client's `connMu`, the atomic closed
flag, connection acquisition/dialing, and `Close`. A get holds the mutex while
it acquires or dials; Close marks the client first, then takes the mutex and
closes any connection. A stream release can independently close the connection.
The model abstracts acquisition and a successful dial as the same operation;
failed dials cannot create a surviving connection. It checks safety, not dial
termination: Close still waits for an in-progress dial's context to finish.

Run with the TLC command above, using `ClientClose.tla` and one of:

| Configuration | Result |
| --- | --- |
| `cfg/close_asis.cfg` | Reproduces #400 in five states: Close finishes, then a get dials a connection that subsequent Close calls do not own. |
| `cfg/close_guarded.cfg` | No error, 9 distinct states: `NoConnectionAfterClose` holds with the closed check under `connMu`, including a dial overlapping Close. |

The generated-module tests `TestClientClosePreventsNewStreams` and
`TestClientCloseDuringDial` exercise these cases against real sockets under the
race detector. Closing before the first dial and after an active stream both
reproduced unwanted redials before the guard was added.
