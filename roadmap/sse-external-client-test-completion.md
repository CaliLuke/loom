# External SSE client test completion ownership (#933)

1. Reproduce the CI false timeout with a controlled callback-before-waiter schedule; retain the original real SSE test and its protocol assertions.
2. Model the current competing event/cancellation completion paths and a single observer owner. Confirm the current counterexample before implementation.
3. Share one private collector across decode-error, oversized-error and ordinary external-client tests. Callbacks collect and signal completion; the observer cancels and joins the connection, then verifies collected events before classifying connection errors or deadlines.
4. Add deterministic helper controls for completion-before-wait, EOF-after-final-event, premature close, deadline and cancellation cleanup. Run focused races and bounded repetitions plus the three actual generated-server tests.
5. Freeze the exact patch and evidence for independent review; root integrates one atomic commit. No generator or runtime change.

Parent: `65a75387b619c99fcab3b365d1653d64c8910328`.
CI witness: run 38597036619, job 115966310525, `TestJSONRPCSSEExternalClientReceivesDecodeErrorsOnMessageEvent` reports timeout after 1.57 seconds with a three-second deadline. The callback publishes an event then cancels the request context; the waiter can select cancellation even with the event queued. Ordinary and oversized-error external-client tests share the same competing completion pattern.

## Progress and acceptance

- Reproduced: CI failure, controlled actual-server schedule (6/10 failures), and current-owner model counterexample.
- Implemented: shared collector owns cancellation and joins the connection before classifying a complete event snapshot; all three test paths use it. Event counts, envelope fields and error codes remain asserted. Unexpected connection errors remain visible.
- Verified: 100 race repetitions of deterministic helper controls (2.028 seconds); all three actual external SSE test paths pass three race repetitions (10.036 seconds). Package vet passes. Corrected model passes 26 generated / 13 distinct states and completion.
- Pending: independent exact final review and atomic root delivery. This changes test completion ownership only; production and generated fixture bytes are unchanged.
