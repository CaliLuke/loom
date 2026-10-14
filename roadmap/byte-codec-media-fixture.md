# Byte decoder media fixture adoption (#914)

The shared request decoder already accepts valid structured JSON media types,
as authorized and delivered in #787. One byte-codec fixture still sends raw
`hi` for `application/vnd.bytes+json` and expects unsupported media. It must
instead send the existing JSON byte representation and assert decoded bytes.

1. [x] Reproduce the exact current failure and check the documented media rule.
2. [x] Change only this case's codec from unsupported to JSON; retain all other
   codec cases, unsupported suffix assertions and byte-value assertions.
3. [x] Run the direct byte decoder/encoder matrix and existing request-media
   tests under race detection. No production or generation change is involved.
4. [ ] Independent exact review, atomic commit/push and issue closure.

Evidence: `/tmp/loom-ci-current-targeted-residuals.log` and CI job115639900616.
This adopts existing behavior; it does not introduce a compatibility exception.

The byte encoder/decoder matrix passes under race detection (1.670 s). The
first command selected no runtime media tests; the corrected focused invocation
ran `TestRequestJSONMediaBodyContract` and `TestRequestJSONMediaLimit` successfully.
All unsupported-suffix and exact decoded-byte assertions are retained.
