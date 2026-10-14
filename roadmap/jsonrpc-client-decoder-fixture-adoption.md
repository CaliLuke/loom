# JSON-RPC client decoder fixture adoption

Issue: #916. Parent reproduction: `02cf3826578f6a03fbf805f9279847368fec63cd`.
This changes tests and their expected generated output only. The shared method
error decoder remains the production owner.

## Sequence

- [x] Reproduce both stale expectations on the recorded parent (0.552s).
- [x] Assert delegation and the unmapped raw-error fallback while retaining HTTP
  mapping exclusion and viewed-result constructor error propagation.
- [x] Regenerate only the owning viewed-client golden; inspect the exact diff.
- [x] Run the two direct tests and existing generated typed/raw error round trip.
- [ ] Obtain independent review of the exact final diff before one atomic commit.

Acceptance: the viewed golden changes only its error dispatch and adds the shared
method decoder. HTTP-only error mappings remain absent from JSON-RPC output;
unmapped wire errors retain their raw fallback. Existing result validation and
fallible viewed-result construction remain checked. The generated round trip
checks both typed and raw errors using the current framework source. No new
formal model or generator corpus is needed for this expectation-only correction.

## Validation and review state

The two original tests failed on the recorded parent. The owning golden was
regenerated with `TestJSONRPCClientDecoderPropagatesViewedResultConstructorError
-update` (0.548s). Its only differences are shared error delegation and the raw
fallback helper. The two repaired tests, the existing unmapped-error structural
control and `TestJSONRPCGeneratedTypedAndRawErrorsRoundTrip` pass together with
`-race -count=1` (3.724s), using explicit local framework source. The generated
round trip compiles and exercises both clients against a mounted server.

Formatting and `git diff --check` pass. Final review parent is
`d1e8c68ef52ad7023a7306aa72df699305134721`; the intervening #914/#915 commits
change unrelated fixtures, not these tests or their production owners. No
production, design or transport contract changes are included. Independent
review and delivery remain pending.
