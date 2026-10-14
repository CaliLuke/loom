# Sink missing-group recovery model

`SinkGroupRecovery.tla` models one active registration with one removal/re-add,
a close, one external group deletion followed separately by membership reset, and
one transient recovery failure. Epochs retain distinct authored starts (7 and 11).
The small domain explores read and rotation entry points without modeling payloads.

| Configuration | Intended observation | States generated / distinct |
| --- | --- | --- |
| current | NoHotRetry fails: repeated NOGROUP has no wait or repair | 19 / 14 |
| snapshot | NoRemovedRevival fails: captured registration creates after removal/close | 170 / 124 |
| main-start | OwnStart fails: fallback selector replaces registration intent | 69 / 55 |
| owned | Safety and Recovery pass | 1212 / 651 |

Run each from this directory with pinned TLC 1.7.4 tools:

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config owned.cfg SinkGroupRecovery.tla
```

The modeled parent corresponds to Sink.read delegating NOGROUP to handleReadError and then
immediately retrying. Snapshot recovery and default-start recovery are rejected
alternatives, not current implementation. Candidate BeginRepair and creation
steps hold the same local lifecycle mutex that Add/Remove/Close use. Remove is
therefore before or after recovery, never inside it. CreateGroup preserves an
existing group's cursor; membership publication must occur first. Pending repair
survives transient failure, including after group creation but before consumer
confirmation. The concrete implementation must use the single reconciler from
read, rotation and existing keepalive; cleanup-only maps are excluded.

DestroyGroup and DestroyMembership may interleave with recovery independently.
ObserveReset abstracts eventual local replica observation and existing keepalive
or read-preflight progress; it is not synchronous distributed reset detection.
`Recovery` establishes eventual stable registration if it remains active and open,
under weak fairness of retry/reconciliation steps and the finite disruption bound.
It does not claim continuous membership while Destroy is running, a time bound,
Redis availability, arbitrary repeated failures, or fairness of network I/O.
`Wait` abstracts one positive interruptible retry delay; tests must check actual
command pacing. No PEL preservation, exactly-once delivery or deleted-event
recovery is modeled. Native tests establish selector command semantics; distinct
model starts establish identity only.

Parent implementation probes are the five TestMissingGroup cases in the readiness
input: three negative cases expose read spin, pre-read rotation failure and delayed
membership reset; two controls check removed registration and initial override.
Acceptance extends existing streaming tests with delayed command barriers rather
than introducing a generic coordination harness. Existing SinkAddStream and
SinkMapOwnership models retain their membership-first and cleanup-handle scope.
