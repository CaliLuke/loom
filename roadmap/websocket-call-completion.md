# WebSocket client call completion — #739

Parent: abca0b975. Delivered and pushed as `ec2536d3c`; independent review and re-review passed, and #739 is confirmed closed.

## Contract and owner

A response routed to a call is held until consumed or the stream ends. Timeout duration still starts at registration; a timer becoming ready only proposes timeout when an active Call/Recv observes it. Do not restart the clock at Recv or introduce background request abandonment. Recv context cancellation preserves the call at the queue head; Call cancellation abandons its call. Stream termination invalidates future operations while other streams remain independent. Concurrent terminal events arbitrate once; completed responses and connection errors cannot be replaced by a later timeout.

The existing connection mutex already owns registration/removal. Extend that owner to publish the immutable terminal result in the same critical section. Use one private locked completion path for routed response, connection read failure, observed timeout, Call cancellation, write failure and stream termination. Each winner removes the registration, stops its initialized timer and publishes one buffered result. Later proposals consume the winner rather than discard or overwrite it. Keep callbacks outside locks, stream-before-connection lock order, and no per-call goroutine or extra allocator/state machine.

Initialize the call timer before publishing registration under the connection lock. On parent abca0b975, registration unlocked before timer assignment, so stopping a timer from response routing without fixing publication creates a nil/race boundary. The call result channel remains the terminal storage; do not add a parallel outcome store. Connection-map membership establishes pending versus completed. Stream cleanup must also publish a terminal result, so losing timeout/cancellation arbitration always has a result to consume; it cannot block on a silently unregistered call.

Recv cancellation is not a completion proposal: requeue the same call only while the stream remains active, preserving a result completed concurrently. Explicit Close/context/write failure keeps current preflight error classification; an already waiting operation may return an earlier completed result, as the existing CloseErrors model permits. Send still reports its write error. Error handlers report timeout only when timeout wins, and protocol errors only when consumed.

## Parent defects and rejected approaches

- A buffered result and expired timer are independent select cases; timeout can discard the already held result.
- Routing removes the registration, unlocks, then publishes. A nonblocking result-priority check alone misses this remove/publish interval and can still return timeout.
- Stopping the timer only in routing leaves ready timeout selection and registration-before-timer initialization races.
- Automatic timeout callbacks or arming on Recv change the existing timeout origin/observation contract and are unnecessary.

The existing WebSocketClientDemux model already allows RecvTimeout to erase `delivered`; its NoLostResponse invariant only covers orphan routing. Extend its owning file with a focused completion refinement and small configurations rather than rerunning the unrelated 16-million-state routing audit.

## Sequence and bounded acceptance

- [x] Add a failing delayed-Recv test using the existing scripted server and direct completion/lifetime controls; record parent results.
- [x] Model current independent selection, result-priority-only and atomic completion, including publication/timer initialization, cancellation/requeue, connection failure and stream end. Record assumptions and counterexamples.
- [x] Obtain independent readiness review of owner and contract.
- [x] Implement the shared completion path and remove superseded independent removal/timer/result policy.
- [x] Run focused WebSocket client race/closure tests and bounded completion configurations; inspect all registration/completion consumers. No generator changes or broad fixture sweep.
- [x] Freeze exact final diff for independent review; deliver only after approval.

Controls: buffered success and protocol error survive delayed Recv; buffered connection failure retains its cause; pending calls time out from registration and late replies are orphaned; canceled Recv preserves send order/result; canceled Call releases pending state; stream Close/context end releases all calls/timers and preserves other streams; route/timeout and fail/timeout cannot both win. Existing real-socket demultiplexer, error-classification and generated pending-order tests remain the consumer surface; run only tests invalidated by the runtime repair.

## Readiness evidence

Parent tests fail under the race detector (1.472s): 18 of 32 actually delivered responses were returned as timeouts, and delivery left its request timer active. Existing `TestWebSocketClient*` controls pass under race (1.626s); scoped jsonrpc lint reports zero issues. The existing model now has focused current-held and split-publication counterexamples; result-priority-only still fails, while atomic completion passes 174 generated / 132 distinct states. One-stream/one-send routing safety controls pass 2,396 / 744 states. These are safety checks under the documented model assumptions, not proof of Go scheduling or new deadline semantics.

Implementation acceptance includes a deterministic direct owner race: response or connection failure that has claimed completion wins against timeout; timeout that completes first unregisters and makes the later response orphaned. Timer initialization must precede registration publication, including an immediate response or concurrent connection failure. Cancellation/requeue and stream termination must neither republish nor silently remove a call that another terminal proposer needs to consume. Reuse existing generated `TestJSONRPCWebSocketRecvAfterResponseGeneratedModule` only if its runtime boundary is invalidated; generator artifacts should remain unchanged.

## Implemented owner and checks

`WebSocketClientConn.completeLocked` owns every terminal registration removal, timer stop and buffered result publication. Route and connection failure consume it under their existing lock; timeout, Call cancellation and stream end use its locked entry point. Stream cleanup publishes each pending error before clearing local ownership. Canceled Recv requeues only while active. There is no added per-call goroutine, lock or duplicate outcome store, and the previous unregister/complete/forget paths are removed. The superseded stream wake-up channel is also removed; cancellation tests observe the actual locked lifecycle state, while requeue admission derives from existing closed/ended/error state.

Focused runtime race checks pass (2.165s). The existing generated pending-order client module passes (5.338s), including vet and its race-enabled runtime controls. No generator code or fixture output changes are intended. The owned model is unchanged from independent readiness approval; reuse its 174/132-state result and bounded original-routing control. Final lint initially caught one unnecessary test conversion; the type declaration was corrected mechanically.

Final affected-package race gate passes (2.221s) and scoped lint reports zero issues. Exact final sources are recorded in `/tmp/loom-739-final-manifest.json`; aggregate CI success is not claimed.

Final-review LOWs 739-R1/R2: removed the obsolete stream done channel/flag, derived requeue admission from existing lifecycle state, and made parent-only descriptions explicit. Both cancellation tests now observe terminal ownership under the stream mutex. Affected WebSocket race checks pass (2.075s), scoped lint is clean; unchanged model and generated evidence is reused.

Independent re-review confirmed both fixes; the integration preserved all ten
functional/model/plan sources and composed only framework guidance. Final root
WebSocket race checks pass (1.964s).
