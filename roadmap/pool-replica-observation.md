# Pool release replica-observation test — #846

Accepted direction: preserve `releaseTrackedJob`'s authoritative payload contract
and make its test wait for the replica it actually reads. No production ownership,
publication, timing or synchronization change. #687/#704 construction work remains
separate. Parent is the pushed `03e38fd16` source containing the failing test.

## Status

- Main CI payload assertion failed after earlier release checks succeeded.
- A test-only payload-subscriber scheduling overlay reproduces that assertion while
  immediate Redis HGET proves the payload is retained. Twenty ordinary repeats pass.

## Milestones

### Milestone 1: Assert durable release and each replica's own completion

Goal

Correct `pulse/pool/requeue_lost_test.go:TestRebalanceRelease` without weakening
payload retention, stopped-handler, index removal or repeated-release behavior.

Acceptance Criteria

- Authoritative Redis payload equals the original bytes immediately after release
  and again after the repeated no-op release.
- Local payload availability is checked eventually on its own replica; worker job
  and index removal checks remain unchanged.
- The deterministic replica-delay witness passes, normal/race/real-Redis targeted
  checks pass, and production files remain unchanged.

Checklist

- [x] Read AGENTS.md, loom-framework and this plan; recite test-first sequence,
  durable/replica distinction, focused gates, independent review and root delivery.
- [x] Preserve the CI excerpt and deterministic witness at
  `/tmp/loom-ci-payload-replica-overlay.json`; ensure the probe changes only payload
  subscriber scheduling and adds a durable assertion, leaving production files intact.
- [x] Change `TestRebalanceRelease` to assert Redis HGET bytes directly, wait for
  `node.JobPayload`'s expected bytes with the existing eventual deadline, and repeat
  the Redis assertion after idempotent release.
- [x] Rerun the same delay overlay with only its original test instrumentation
  removed, confirming the permanent test passes under the identical delayed schedule.
- [x] Run `GOEXPERIMENT=jsonv2 go test -race ./pulse/pool -run '^TestRebalanceRelease$'
  -count=20`, then the same named test against the established pinned local Redis.
- [x] Run scoped pool lint and `git diff --check`; preserve a compact receipt.
- [ ] Freeze the exact atomic diff for independent review and root commit/push;
  close the issue only after successful delivery and clean inactive temporary probes.


## Acceptance receipt

Issue: https://github.com/CaliLuke/loom/issues/846. The original CI failed the
payload-availability assertion, not the earlier eventual job-index removal.
The bounded replica scheduling overlay delays only this test's payload-map
subscriber before it applies the update. The original test fails its same
predicate while Redis HGET returns exactly `payload`
(`/tmp/loom-ci-payload-replica-probe.log`). The pool test and replica implementation
are byte-identical between that probe's `f4bfd3c38` parent and this worktree's
`03e38fd16` parent.

The corrected test passes with the identical 100 ms subscriber delay and no
extra test instrumentation: `/tmp/loom-pool-replica-delay-green.log` (1.067s).
Only the observation boundary changes: durable bytes are asserted synchronously,
and the independent payload replica is allowed to finish its own update. Existing
stopped-handler, empty local jobs, removed job index and second-release no-op
checks are retained; a second durable-byte check follows that no-op.

Twenty miniredis race runs pass in 13.952s
(`/tmp/loom-pool-replica-race.log`). The same named test passes with race enabled
against the existing loopback Redis 8.10.2 container in 2.460s
(`/tmp/loom-pool-replica-redis.log`, image `/tmp/loom-846-redis-image.txt`), using
only disposable DB3 through the established helper. Scoped pool lint reports zero
issues (`/tmp/loom-pool-replica-lint.log`); `git diff --check` passes.

No production file, generator, timeout value or pool protocol changes. A new
formal model is unnecessary for this test-observation correction; the existing
ownership models and runtime implementation remain unchanged. Exact independent
review, root integration and atomic delivery remain pending.
