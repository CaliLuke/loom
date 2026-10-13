# Mixed HTTP/SSE fixture freshness — #881

The independently inspected CI reproduction on pushed `055c2eb0a` shows that
12 of the mixedtick fixture's 16 generated files are stale. Five integration
scenarios stop at full-tree parity before transport assertions. The changes
reflect previously delivered ID binding, client stream context/signatures,
response projection and optional diagnostic reporting. This is fixture drift;
no new generator behavior is proposed.

1. [x] Preserve the failing tree comparison and all parent/generated differences.
2. [x] Separate the stale params-error scenario as #882; four other mixed
   scenarios pass with a fresh temporary fixture.
3. [x] Regenerate the checked-in mixedtick fixture with the current local CLI;
   retain the authored design and full-tree comparison.
4. [x] Compare all artifacts against an independent generation, build and vet
   the complete fixture, and run its four unaffected transport scenarios.
5. [x] Obtain independent exact-diff review, commit, push and close #881.

The params-error scenario must remain unchanged in this atomic repair, with its
known #882 failure recorded. Do not weaken the parity assertion or claim the
whole CI suite is green. Use the existing fixture and integration harness;
there is no new protocol or ownership model to explore. Record exact source,
artifact differences and validation evidence before review.

Final parent: `e631f0a89`. Regeneration changes exactly 12 of 16 artifacts;
fresh-process generation matches every candidate byte. The six authored fixture
files, including module manifests after tidy, are unchanged. Complete fixture
`go build ./...` and `go vet ./...` pass. The four unaffected mixed integration
scenarios pass in 11.766s and again reach their transport assertions. #882 remains
separate and unresolved in this diff.

Intended differences are the already delivered service stream Close contract;
CLI envelope-ID flags/builders; request body exclusion of envelope IDs and ID
conversion at decoding; client notification/body ownership and stream context;
server selected-result projection and optional failure reporting; and generation
manifest refresh. No framework generator/runtime implementation changes here.
Evidence is retained in `/tmp/loom-881-artifact-comparison.json`,
`/tmp/loom-881-controls.log`, and the baseline CI triage packet.

## Delivery receipt

Delivered as `bbe2008ac`, independently reviewed, pushed and confirmed closed. Exact review: `/tmp/loom-881-independent-final-verdict.json`. Separate #882 subsequently restores typed params coverage.
