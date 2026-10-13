# Shared test-module view assembly

The JSON-RPC and mixed HTTP/JSON-RPC test-module renderers omit required service
view files. Seven callers manually render those files afterward. Centralize
assembly in the two renderers, as the HTTP renderer already does; this is a test
infrastructure correction, with no production or wire-contract change.

1. Add a regression using the existing nontrivial viewed unary fixture through
   both shared renderers. Add a plain HTTP endpoint to the mixed case. Require
   the emitted views package and build/vet the resulting complete module. Record
   the missing-file failure before changing either renderer.
2. Add service view files alongside service and endpoint files in the shared
   assembly loops. Remove all seven redundant caller-local loops and imports;
   preserve their existing assertions and runtime harnesses unchanged.
3. Capture bounded pure/mixed viewed and ordinary module outputs before and after,
   then repeat generation in a separate process. Expected new files are the view
   packages missing from the shared helper; existing output must stay identical.
   Run the direct regression and the affected existing viewed/bytes/recursive/
   WebSocket checks, plus scoped lint and diff whitespace checks.
4. Review the report and exact final atomic diff independently. Commit and push,
   close only the proven report, then remove #847's uncommitted caller workaround
   when advancing that separate ticket to the delivered helper owner.

- [x] Reproduce and review the missing-view report.
- [x] Centralize assembly and remove caller compensation.
- [x] Bounded output, compile/runtime and lint acceptance.
- [ ] Fresh exact-diff review, atomic commit/push and confirmed closure.

Acceptance for #858 at parent `56c46446f99c510788931c597e78836244eee838`:
new direct tests fail on the parent for both missing view files; ordinary control
passes. Candidate three-module build/vet passes (3.028s). Six existing viewed,
bytes, recursive and WebSocket tests pass (17.981s), and both local-name module
sweeps pass (18.929s). Scoped JSON-RPC codegen lint reports zero issues.
Across pure, mixed and ordinary fixtures, the parent helper emits 58 files and
the candidate 60: only the two required view files are added. All 60 files match
the prior complete caller assembly byte for byte, and independent-process repeat
matches the candidate exactly. These are test-module assembly checks; production
and wire behavior are unchanged. Hash evidence is retained in
`/tmp/loom-module-views-artifact-hashes.json` pending review and cleanup.
