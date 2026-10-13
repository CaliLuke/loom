# WebSocket message-limit close observation — #825

Base: `5bb31fd1910eb17c7fa1ba8256b201f4d1b0fc9c`.

1. [x] Reproduce the CI socket-release failure on current source with a bounded repeated generated test.
2. [x] Establish whether the failure is runtime ownership or an observation ordering defect. Model close-frame delivery and native closure as separate events; preserve a deterministic regression if ordering is the cause.
3. [x] Repair the owning layer, keeping message-limit, close-code, invocation and socket-release assertions meaningful.
4. [ ] Run focused race/compile checks and generated-output equality if generation remains unchanged; obtain fresh independent exact-diff review.
5. [ ] Root commits one atomic ticket, pushes, closes only with passing evidence and updates the master checkpoint.

Observed CI symptom: after the remote client receives close code 1009, the test immediately checks the server socket with SetDeadline and sometimes gets nil. The close-frame write precedes native Close in the shared lifecycle owner. Receiving the frame may not establish that native Close has completed. The repeated and gated reproductions below confirm this is a test observation defect.

The repair changes test observation only; runtime and generated output remain unchanged.


## Acceptance and evidence

Scope is test observation only. The gated old assertion fails deterministically;
the actual runtime preserves required close-code/error/invocation behavior.
The corrected harness must observe native Close completion within its context,
then keep the original net.ErrClosed assertion. Never weaken socket release to
peer-frame receipt or induce it with polling.

- Parent repeat witness: `/tmp/loom-ws-close-repeat.log`, 6.394s, generated100
  iterations fail only server/large=true socket check.
- Gated parent: `/tmp/loom-ws-close-gated-red.log`, 3.394s, same failure in one
  iteration. Gate-only patch is recoverable from the new harness by removing
  its post-receipt observation/release block, leaving deferred gate cleanup.
- Candidate ordinary run passes; two independent generations with100 race
  runtime iterations pass4.746s/4.548s. `/tmp/loom-ws-close-candidate-repeat100.log`
  and `/tmp/loom-ws-close-repeat100.log` retain commands/results.
- `/tmp/loom-825-comparison.json` records identical parent/candidate/repeat
  generated-byte hashes. No intended generated-output differences.
- Scoped golangci-lint reports0issues. Bounded model results and assumptions
  are in `jsonrpc/codegen/tla/close_observation/README.md`.
- Fresh independent final review remains required before root commit/push.
