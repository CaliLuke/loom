# HTTP collection expectation adoption (#897)

The shared #889 converter preserves absent collection storage, allocates present
empty collections and clears existing targets when their source is absent. The
HTTP body, CLI and JSON-query consumers already use that owner. Their remaining
saved expectations must adopt that contract without changing production code.

## Sequence and acceptance

- [x] Read the owning conversion and affected tests. Reproduce current-source
  failures before edits: `/tmp/loom-ci-current-http-selected.log` records the
  same families as source `2a98e4123` CI run `38473286562`.
- [x] Regenerate the 32 failing snapshots listed in
  `/tmp/loom-897-golden-paths.json` using existing test generation. Review every
  difference: only nil-source allocation guards and absent-target clearing are
  intended; element conversion, validation and request framing stay identical.
- [x] Adopt the same guards in `TestOptionalValueRequestBody` and
  `TestRequestDecoderMapQueryPresenceIsScopedToMapParam`. Keep required-body
  admission assertions separate from collection-storage conversion guards;
  preserve optional-body/CLI behavior and named-query scoping assertions.
- [x] Run the affected comparison tests under race without update mode. Run
  existing root/nested/named collection converter tests and
  `TestOptionalValueRequestBodyGeneratedIntegration` plus
  `TestDeepObjectReadinessGeneratedWire` for compiled body/CLI/query consumers.
  These cover actual absent/empty/value behavior, decoding failures and invocation.
- [ ] Obtain independent exact-diff review, commit/push this one adoption ticket,
  close #897 and preserve compact reproduction/review evidence before cleanup.

Workflow: establish the red expectations first, update their shared contract
adoption, run the named focused checks, freeze the exact diff for an independent
reviewer, then deliver atomically. No production edits, new dependencies,
consumer policy changes, generated `gen/` edits or exhaustive corpus runs.
The existing #889 model and proof sources are unchanged and their evidence is
reused; these test-only edits introduce no new concurrency or protocol owner.

Validation on `87633842f` (only checkpoint documents differ from the original
red source): all selected snapshot and structural comparisons pass under race
(11.971s), compiled root/nested/named converter controls pass (10.085s), and the
two existing generated transport modules compile, vet and pass their runtime
checks (9.934s). Production sources are unchanged. All 32 snapshot differences
are source-presence guards, explicit nil clears and their indentation; no element
conversion, validation, query key or framing changes occur. The streaming
constructor snapshot was captured exactly from the existing test's freshly
rendered output, since that contains-based fixture does not support `-update`.
Configured lint on the changed package reports zero new issues.
