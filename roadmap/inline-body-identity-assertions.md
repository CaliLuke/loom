# Inline body type-identity assertions — #849

Parent: `22d000aa8ce70a8f5d1fe72354d0963be240e29a`.
Scope: test-only correction; no generator, design or serialization policy changes.

1. Reproduce the stale substring assertions while preserving proof that the full
   generated module compiles. Inspect separately named request/client and
   response/server declarations and their transform functions.
2. After independent report/readiness review and filing, replace whitespace/text
   counts with parsed AST comparisons. Select SubmitRequestBody with
   NewSubmitRequestBody on the client and SubmitResponseBody with
   NewSubmitResponseBody on the server; include Holder body declarations and
   their named transform helpers for nested named-object coverage.
3. Collect complete anonymous struct shapes from each selected declaration and
   its selected transform, preserving field order, names, full type expressions
   and exact unquoted tag values. Compare actual emitted anonymous composite
   literal types against the declaring owner's shapes. Assert nonempty expected
   shape coverage so a skipped or missing target cannot vacuously pass. Repeated
   equal collection element shapes are one identity; compiling the complete
   module checks each concrete assignment and nesting.
4. Add bounded negative controls to the structural comparison: mutate one literal
   tag, mutate a field type, and remove a uniquely shaped expected literal. Each
   must fail independently while unchanged parsed output passes. Do not create a
   new tag goldens policy in this test.
5. Keep the existing DSL bytes unchanged and retain generated full build; add vet.
   Compare all generated files before/after and a fresh process for this one
   existing fixture. Every byte must match; no production output is intended to
   change. Run the named HTTP regression plus the existing JSON-RPC inline-body
   compilation control, affected lint and independent exact-diff review.
6. Root integrates one atomic ticket. Preserve reproduction/hash summaries and
   remove inactive generated output after review.

Evidence: root `/tmp/loom-ci-22d-regressions.log`, generated type snapshots
`/tmp/loom-inline-parent-client.go` and `/tmp/loom-inline-parent-server.go`, and
independent reproduction `/tmp/loom-ci-owner-readiness-red.log`. Actual normalized
JSON tags differ from the old strings while declarations/literals agree and
`go build ./...` succeeds. The comparison owns type identity; shared tag behavior
is already covered elsewhere. No new formal model is needed for a linear
assertion-only change; generated compiler and targeted negative controls check
this test's correspondence directly.

Proposed source scope: existing HTTP inline-object test, a test-only AST helper
if separating it improves readability, and this plan. No JSON-RPC production or
fixture edits are required. The report draft is `/tmp/loom-inline-identity-issue.md`.

- [x] Identify stale assertion owner and actual generated contract.
- [x] Reproduce current-parent failure and successful generated build.
- [x] Independent report/readiness review and filing (#849).
- [x] Structural repair, negative controls and byte-identical verification.
- [ ] Independent final review and atomic integration.

Validation completed: the HTTP and existing JSON-RPC inline compilation tests plus
five assertion controls pass under the race-enabled framework test runner
(HTTP 3.335s; JSON-RPC 2.524s). The HTTP generated module builds and vets.
Affected HTTP codegen lint passes. The original fixture DSL is unchanged, and
all 12 generated files match byte for byte across parent assertion code, repaired
assertion code, and a separate generation process. Production generator source
is unchanged; the repeat captures output without repeating build/vet.

Evidence: `/tmp/loom-849-final-tests.log`, `/tmp/loom-849-lint.log`, and
`/tmp/loom-849-artifact-hashes.json`. Parent comparison still fails the stale
assertions after its successful module build; the repaired comparison passes.
Final review and integration remain pending.
