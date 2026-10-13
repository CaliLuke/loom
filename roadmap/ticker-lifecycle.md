# Ticker local lifecycle — #738 and #707

Parent: c01bd544715e3487ddcbc1d12a8c263dee410589. The family design is approved. #738 local ownership is implemented and awaiting final review; #707 remote observations remain the next separate atomic repair.

## Contract and owner

A ticker owns its local subscription, child context and timer. Construction owns these until it returns a live ticker; failed construction releases them. Close stops only this participant. Stop requests deletion of the distributed entry, then stops locally. Remote absence, whether observed through a map notification or authoritative compare-and-set, ends the local participant. Terminal participants cannot publish ticks or rearm a timer; transient Redis advance errors on live participants remain retryable.

Use one private idempotent local teardown transition under the ticker lock. Keep waiting for the event goroutine outside that transition so the goroutine can use it without waiting on itself. Route constructor abort, Close, Stop, map-channel closure, runtime cancellation and missing shared entry through this owner. Borrow the node's map; never close/destroy it from ticker cleanup. A failed SetAndWait may have committed: abort must not delete the shared entry or affect a peer.

A timer event may already be selected when Close begins. Check terminal state under the same lock before advancing/rearming/sending. CAS absence is represented by TestAndSetEx's existence result, not by deserializing an empty value. Preserve the existing successful CAS winner and mismatch retry behavior.

## Sequence

- [x] Reproduce #738 constructor subscription/context retention and #707 missing-entry panic using existing rmap/miniredis helpers; add direct local-close/queued-tick and healthy controls.
- [x] Compare current behavior, constructor-only cleanup, missing-entry-only handling and shared terminal ownership in a small model beside existing pool lifecycle models. Explicitly cover delayed notification, transient retry and local/shared distinction.
- [x] Freeze readiness and receive independent owner approval (`/tmp/loom-ticker-root-readiness-verdict.json`). Establish atomic boundaries without parallel cleanup paths.
- [x] Implement #738 constructor/Close/Stop ownership and all timer admission guards; run focused direct/race controls and the local model.
- [ ] Obtain fresh #738 final review and deliver.
- [ ] After #738 delivery, implement #707 remote consumers, rerun its saved reproduction and full model, then review and deliver.

## Bounded acceptance

Failed constructor releases subscription/context on every attempted call. Healthy ticker retains them until Close; repeated Close is safe and does not delete shared state. Remote deletion before timer processing cannot panic or leave owned resources alive. A selected timer event after local Close cannot rearm or send. Transient Redis failure while live rearms; a healthy peer remains usable after local Close or failed construction. Existing TestTickerDeliversTicks, TestScheduleRunsAndStops and scheduler local/global stop controls remain controls.

No stream reader/sink is acquired, so #800 does not block this owner. This scope does not change scheduler context semantics, producer job tracking (#737), worker publication (#704), node acquisition rollback (#687), malformed externally written ticker values, or tick interval validation. Existing pool ownership and ShutdownClose models cover different states and cannot establish this local timer/subscription contract; their model directory and local/global semantics are reused without rerunning broad state spaces or Redis suites.

## Proposed atomic implementation boundary

1. #738 establishes the local owner: constructor abort/transfer, shared locked teardown used by Close/Stop, terminal tick admission and shared timer-rearm guard. Keep the existing global-delete ordering. The `local-owned` model checks this limited slice; constructor failures, repeated local/global close and preselected work are its direct acceptance. Preserve the #707 failing repro as pending readiness input rather than committing a failing regression in #738.
2. #707 derives remote termination from that owner: authoritative CAS absence and event-loop exits use the same transition (without self-wait), and absence is never deserialized. The full `owned` model adds these remote observations. Deliver its saved repro after it passes. No second cleanup mechanism or timer guard is introduced.

This is one family design with two coherent atomic deliveries, not two independent patches. A private terminal bit records released ownership even when map subscription or timer is nil. The shared rearm helper must refuse work after that transition; guarding only handleTick leaves preselected map updates able to rearm.

## Readiness evidence

`/tmp/loom-ticker-parent-red.log` reproduces five failing cases with race enabled (1.162s): three canceled attempts retain three subscriptions/context registrations; committed-write/lost-reply retains failed ownership while the peer and shared value survive; CAS absence panics; selected tick and selected notification rearm after Close. The notification test targets the shared rearm seam because the event loop reads a map value before taking the ticker lock. Existing native map operations, not fake distributed storage, establish the mutation outcomes.

`/tmp/loom-ticker-controls.log` passes focused local/global close, live transient retry and existing Tick/Schedule stop controls with race enabled (2.423s). No broad Redis suite or generator work was run. The new tests use a minimal Node containing a real rmap and a standards-compatible context AfterFunc counter, avoiding unrelated stream startup/teardown.

No promise of forcibly interrupting Redis commands is introduced. The model treats each Redis command outcome as atomic and assumes it returns under the existing client limits; it proves terminal ownership safety, not network-timeout bounds, fairness, tick delivery liveness or distributed clock scheduling. Local map notifications can lag authoritative mutations, and explicit close may race already-selected work.

## #738 implementation

`stopLocalLocked` owns exactly-once local teardown; Close waits only after releasing the lock. NewTicker uses one deferred local Close until successful transfer. Stop performs its existing shared-delete attempt before local cancellation; even a failed delete releases local resources without claiming shared deletion succeeded. handleTick rejects terminal work before native I/O and resetTimerLocked rejects every late rearm. Borrowed map/caller client and uncertain committed shared values remain untouched.

The #707 direct missing-entry test is preserved externally at `/tmp/loom-707-pending-repro_test.go` until the next atomic repair. Its parent evidence and full candidate model remain in the readiness manifest; no failing remote regression is included in #738's delivered test file. The full `owned` model remains a proposal for #707, whereas `local-owned` corresponds to this implementation.

Final #738 checks: focused ticker and scheduler consumer controls pass with the race detector (2.908s); scoped `golangci-lint run ./pulse/pool` reports zero issues. TLC `local-owned` passes with 39 generated / 15 distinct states. The saved #707 overlay still fails with the original absence panic and retained local ownership (0.711s), confirming that this packet does not claim remote termination is repaired. No generator or broad Redis suite is relevant to this private runtime-owner change.
