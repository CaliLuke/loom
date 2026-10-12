# Forwarded client trust readiness

Status: independently approved by `/root/review_artifacts` on 2026-10-12.
Owning ticket: #708. Root releases implementation from parent `23f2af5d`, with
the test-first/model-first sequence below. Record the full parent and assigned
worktree before editing source.
Root read the live report, `http/request_metadata.go`, its direct tests and
`docs/http-guide.md` before preparing this bounded task.

## Contract and owner

`RequestMetadataPolicy` is already the owner of trusted-proxy interpretation.
The direct peer must be trusted before any forwarding header participates.
`forwardedClient` currently removes every unparseable entry, then walks parsed
addresses from right to left. Removing entries loses a trust boundary and can
select attacker-supplied text to the left of an opaque nearest hop.

Keep every entry in the traversal, including empty and invalid ones. Walk raw
header values and their comma-separated entries from right to left. Stop at the
first untrusted parsed address and select it. If parsing fails before reaching
such an address, stop and retain the direct peer as `ClientAddr`. Do not inspect
entries farther left. An absent header or an all-trusted chain retains the
existing direct-peer fallback. Preserve `PeerAddr` and the separate host/proto
handling. Do not add support for port-bearing or obfuscated entries in this
ticket; the existing parser admits bare IP addresses and unmaps mapped IPv4.

This changes the shared snapshot once; logging and endpoint/security accessors
consume that decision. Do not add compensating checks in callers or middleware.
There is no new public API, dependency, generated shape, or application-owned
trust rule. Applications continue to select trusted proxy CIDRs.

## Acceptance matrix

- Trusted direct peer plus attacker prefix and invalid rightmost hop: select
  direct peer. Include `unknown`, `_hidden`, `ip:port`, empty entry and malformed
  bracketed address from the reported failure family.
- Invalid middle hop with only trusted hops to its right: select direct peer,
  independently of the left prefix. This deliberately does not guess a client
  from the last trusted proxy address.
- Untrusted parsed hop nearer than an invalid or attacker-controlled prefix:
  select that hop; farther-left malformed text does not invalidate an already
  established boundary.
- Repeated header lines preserve the same traversal as an ordered comma chain.
  Leading/trailing spaces around an otherwise valid address remain accepted.
- Valid IPv4, IPv6 and IPv4-mapped cases retain existing selection; missing and
  all-trusted headers retain fallback. An untrusted/invalid direct peer ignores
  the header completely. Existing metadata immutability tests remain valid.
- `EffectiveClientAddress` and the logging consumer observe the same selected
  value; there is no raw-header fallback around the snapshot policy.

## Files, reproduction and bounded evidence

1. Add table-driven regression cases in `http/request_metadata_test.go`. Run
   `go test ./http -run 'TestRequestMetadata|TestEffectiveClientAddress' -count=1`
   on unchanged source and record actual failing cases before implementation.
2. Add a small model beside the owner at `http/tla/forwarded_client/` with legacy
   filtering and corrected traversal configurations. Distinguish trusted,
   untrusted and invalid entries, header ordering and a direct-peer fallback.
   The legacy configuration must select a spoofed left prefix across an invalid
   boundary; corrected selection must not depend on entries left of the first
   untrusted or invalid hop. State sequence bounds and keep Go parser behavior
   outside the model's claim. Use the existing pinned TLA tools and external
   state directories, recording exact commands and outcomes in its README.
3. Correct only the shared traversal in `http/request_metadata.go`; remove any
   import made unused by eliminating eager filtering. Extend direct logging
   tests only if existing consumer coverage cannot establish the selected
   snapshot reaches the log field.
4. Run `go test -race ./http ./http/middleware ./clue/log -count=1`, scoped vet,
   formatting and applicable repository lint checks. This is a handwritten
   runtime/middleware correction; no generator source or output changes are
   expected, so a generation sweep is not an acceptance gate.
5. Update the inbound metadata section of `docs/http-guide.md` with the exact
   invalid-hop fallback. Record reproduction, model bounds and passing evidence
   here and beside the model. Send root concise framework-skill/model-index
   deltas; no unrelated header parsing or proxy feature changes.

Freeze the candidate for fresh independent review. One atomic #708 commit must
contain its implementation, regressions and durable evidence. Root integrates,
pushes and updates the master inventory. Remove temporary checker/probe output
only after review and after checking active users.

## Execution record

- Worktree: `/Users/luca/.codex/worktrees/forwarded-client-trust/loom`.
- Parent: `23f2af5d2e4ad9c9b659309839af4f3b25b3f8e3`.
- Implementer plan: add the boundary and consumer regression matrix first;
  model legacy filtering and corrected raw traversal before changing runtime;
  replace only shared traversal; document fallback; run bounded race/vet/lint
  gates and freeze a manifest for root integration and independent review.

### Frozen candidate and validation

The source candidate is ready for root integration and fresh independent review;
no commit or delivery is claimed. The runtime change removes eager address
filtering and walks raw header lines and comma-separated entries right to left.
Only `forwardedClient` and its private call signature change; metadata snapshot,
logging, host/proto selection and public APIs retain their existing owners.

Before changing runtime code, the new 23-case table failed twelve invalid-hop
cases at the recorded parent. The remaining normal, nearer-untrusted and
untrusted-peer controls passed. `go test ./clue/log -run TestFrom -count=1`
also failed the new invalid-boundary consumer case. Both failures selected the
attacker's prefix instead of retaining the direct peer. The corrected source
passes the same direct command and the accessor/logging cases together:

```sh
go test ./http ./clue/log -run 'TestRequestMetadata|TestEffectiveClientAddress|TestFrom' -count=1
go test -race ./http ./http/middleware ./clue/log -count=1
go vet ./http ./http/middleware ./clue/log
make lint-gofmt lint-namescope
git diff --check
```

All these gates pass on Go 1.27.2, Darwin/arm64. Race package times were 4.507s,
1.227s and 1.674s respectively. No generator code, generated artifact, module
manifest or dependency changed; no generated-output sweep was run.

The pre-implementation legacy model fails `BoundarySelection` with a spoofed
untrusted address left of an invalid entry. The checked configuration passes
56,880 distinct states (85,790 generated), covering zero to four hops, direct
peer trust, selected boundary, and same-length arbitrary prefix substitutions.
See [`http/tla/forwarded_client/README.md`](../http/tla/forwarded_client/README.md)
for pinned tool identity, exact commands, correspondence and limitations.
Go tests additionally cover changed prefix lengths, actual IP parsing and
repeated header lines, which are abstracted by the model.

`make lint-json-v2` is blocked by pre-existing prose in
`roadmap/example-interceptor-ownership-readiness.md:152`, which quotes the old
JSON v1 package spelling. The file is identical to parent `23f2af5d` (SHA-256
`781e070b3751e53585b73de7e2e64a7467c8f6bbb034fbbc21d545afe295ddd3`).
The source harness correction from `93f513be` is intact. This unrelated
readiness file remains outside #708; root was notified of the lint blocker.

Scratch for baseline Go logs and both TLC logs/state directories is
`/var/folders/v7/x61rxqvx60q5w_p9jwgc9t_m0000gn/T/loom-708-evidence-vwjoiuu0`.
It is retained for independent review, then must be removed after confirming
no active process needs it. Durable reproduction inputs and concise results
remain in this readiness record and the local model README.

Integration parent is `26f252ef`. Only documentation changed since the recorded
implementation parent. That checkpoint corrects the historical prose lint
blocker; `make lint-json-v2` passes on the integrated candidate. Runtime inputs
and the recorded model and race evidence remain applicable.
