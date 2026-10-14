# Header formatter expectation adoption — #903

This is an atomic test-expectation repair. Production formatting, naming,
transport behavior and authored designs remain unchanged. Implementation parent is delivered `6eb4159e5`, after #760 preservation,
cleanup and safe worktree reuse.

1. [x] Confirm the focused current-parent failures using the preserved issue report
   and `/tmp/loom-ci-206-current-probes.log`; once the clean delivered parent is
   available, compare relevant source identities before deciding whether a repeat
   is needed. Ignore the unrelated alias-route admission rows in that same log.
2. [x] Regenerate exactly two owning TestEncodeError goldens:
   `primitive-error-in-response-header` and `empty-error-response-body`.
   Use `-update` through the existing renderer. Preserve the other twelve cases.
   Intended changes are typed nominal scalar formatting: retain `val := res`,
   then convert at the shared formatter (`string(val)` / `strconv.Itoa(int(val))`)
   rather than pre-casting the source and expecting an old expression.
3. [x] Update only superseded strings in TestTransportVarNamesAreUniquePerFunction.
   The client constructor argument is `tagsSlice`, agreeing with its declaration.
   The server value is already dereferenced before scalar formatting, so expect
   `val := *res.Tag` and `tags := strconv.Itoa(val)` together. The same existing
   server case has further stale expectations hidden by require's first failure:
   converted list storage is `numss` (declaration, allocation and SetHeaderValues
   reference must agree), and the cookie pair is `countraw := *res.Count` plus
   `count := strconv.Itoa(countraw)`. Preserve those obligations with current
   expressions; do not simply remove the assertions.
4. [x] Retain renderTransportFuncs' duplicateScopeVars AST check unchanged. Retain
   the actual TestTransportVarNameCollisionsCompile module, including its vet,
   race execution and transportMapCarrierHarness. This checks naming references
   compile and existing query-map/presence behavior survives. No new harness is
   needed. Run all14 TestEncodeError rows and all11 named collision assertion
   rows, plus that existing generated module. Scoped lint and diff checks apply.
5. [ ] Freeze the exact four-file diff (two goldens, one test, one owning roadmap)
   for a fresh independent review against its intended delivered parent. Confirm
   all generated snapshot edits equal renderer output and no production file
   changed. Let the orchestrator integrate one atomic commit; then preserve
   compact evidence and clean inactive temporary outputs.

No new model or generator audit is warranted: this changes only expectations of
already delivered owners. The shared transportScalarValue/renderHeaderConversion
and existing function-local name allocation remain authoritative. If the focused
compile check exposes a real production defect, preserve it and report it rather
than changing formatting or weakening the test under this ticket.

## Existing evidence

- Issue: `/tmp/loom-903-full-issue.json`, original source b40eb24e6.
- Current focused parent evidence: `/tmp/loom-ci-206-current-probes.log`.
- Actual CI: `/tmp/loom-ci-206-ci.log`, run38484604131.
- Report: `/tmp/loom-header-expectation-adoption-report.md`.
- Owners: `http/codegen/transport_value_conversion.go` and
  `template_sources_response.go`; paired name declarations/references appear in
  the exact rendered functions retained in the current-parent failure log.

The scratch readiness assessment made no source edits. The clean delivered
parent reproduced the same four failing subtests in `/tmp/loom-903-parent.log`
before implementation. #760 is now delivered and closed.

## Final validation

On `6eb4159e5`, the exact focused parent command reproduced two error snapshot
mismatches and the two collision-table failures. Regenerating only the named
error cases changed three scalar conversion pairs across two golden files.
The collision table now checks the actual `tagsSlice` constructor argument and
retains stronger declaration/formatter/publication checks for the scalar and
list locals. `duplicateScopeVars`, the compiled-module setup and its behavioral
harness remain unchanged.

All 14 error cases, all 11 collision table rows and the existing generated
collision module passed together with race enabled (8.970 seconds). The module
also ran its existing `go vet ./...` and race harness. A scratch overlay changed
only its output directory to retain the actual artifacts for review. The two
regenerated goldens matched a second process's ordinary comparison. Scoped
HTTP-codegen lint reports zero issues. No production source changed, so no new
formal model, unrelated output sweep or repeated identical module build was run.

Exact sources, owners, input capture and artifacts are recorded in
`/tmp/loom-903-final-manifest.json`. Delivery remains pending independent exact
review and the orchestrator's single atomic integration commit.
