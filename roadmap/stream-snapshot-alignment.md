# Stream snapshot alignment (#869)

Bring existing streaming expectations into agreement with delivered response projection and fallible protobuf conversion. Keep production unchanged and retain exact-output assertions.

## Sequence

- [x] Reproduce required-gate failures and distinguish the separate #868 compiler defect.
- [x] Regenerate the six HTTP full-file goldens and align eleven HTTP/four gRPC rendered section expectations.
- [x] Inspect every output difference: only body bindings and conversion error propagation are intended.
- [x] Run the three affected test groups and focused structural lint; reuse the already reviewed production/model evidence.
- [x] Obtain independent exact-diff review, commit, push and close the maintenance ticket.

## Acceptance

CI run 38391460881 and `/tmp/loom-ci-866-current-goldens.log` reproduce the same 21 mismatches. The latter ran at pushed 56a78637a; subsequent #859 changes do not modify these generator paths. The complete affected test groups must pass after updating expectations. Test inputs, exact comparison assertions, generators and runtime code remain unchanged; no new model or broad generated corpus is warranted for this test-only correction.

## Validation evidence

The six full-file goldens were regenerated through `TestWebSocketGoldenFiles -update`. Fifteen embedded expectations were replaced with exact actual sections from the failed owning tests, requiring each prior literal to match uniquely. All eleven Go files are byte-identical outside expected-output raw strings. The complete affected groups pass on c8515931c plus this test-only diff: HTTP 0.438s, gRPC 0.513s. gofmt and diff checking pass. Seventeen snapshot files change; the additional file is this plan. No production, test-input or assertion changes are included.

Delivered as `641c103c2`, independently reviewed, pushed and confirmed closed.
