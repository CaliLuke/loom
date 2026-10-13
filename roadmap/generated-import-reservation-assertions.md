# Generated import reservation assertions — #852

Parent inspected: `22d000aa8ce70a8f5d1fe72354d0963be240e29a`.
Scope: replace a generator-source-file heuristic with generated-file collision
coverage. Do not add a production `loomdiagnostics` reservation to silence the
false positive. #850's test assembler repair is a prerequisite for rendering
credential-bearing fixtures through the shared module helpers.

1. Preserve the failing named test and inspect the two distinct emitted files
   represented by `http/codegen/diagnostics.go`. The standalone diagnostic package
   imports no service or views package; the example server's dynamic diagnostic
   imports belong to another generated file.
2. Independently review and file the exact report before implementation. Resolve
   the test-owner scope using the existing `internal/testingx` generated-source
   inspection seam, rather than introducing function-name exceptions.
3. Add a focused generated-file import-name collector beside the existing
   service-package shadow-name collector. Reuse its identification of service
   and views imports. Only collect external import aliases from a file containing
   such an import; exclude generated/module-local imports whose aliases are
   design data. The controlled fixture assumption below identifies the remaining
   aliases as framework-owned. Preserve explicit aliases and default package
   names for the dependencies these fixtures actually emit.
   Test same-file collisions, separate-file non-collisions (including the same
   package), service/views paths, dynamically named module imports, and blank
   imports. No generator-source filename should affect the result.
4. Replace the HTTP source-AST scanner with rendered-file assertions using the
   existing all-functions/WebSocket designs. Require nonempty fixed import
   coverage and compare observed aliases to the existing reservation owner.
   In JSON-RPC, feed observed fixed aliases into the existing generated-name
   compilation sweep alongside the existing local names, so a missed real
   reservation produces generated compilation failure without exporting the
   HTTP reservation table. Inspect costs before adding an independent sweep;
   reuse the existing one.
5. Preserve a focused negative control: an observed same-file fixed alias absent
   from a supplied reservation set is rejected. A standalone `loomdiagnostics`
   file is excluded, but the same alias co-located with a service import remains
   observable. Delete superseded source parsing helpers completely.
6. Run direct collector controls, affected rendered assertions and the existing
   HTTP/JSON-RPC name compilation sweeps. The test-only patch must not modify
   any production generator or reservation list. Reuse unchanged generated
   fixtures and #850 source-assembly evidence; compare bounded rendered output
   where the test input changes, and compile those cases. Do not claim a full
   template-branch inventory from a representative rendered fixture.
7. Freeze the exact atomic diff for independent review, then root integrates.
   Preserve compact reproduction/evidence and remove inactive artifacts.

The existing HTTP all-functions/WebSocket and JSON-RPC HTTP-SSE/WebSocket
fixtures declare no authored external custom types. Under that explicit input
constraint, imports outside the generated application module are candidates for
framework-owned fixed aliases. Rendered syntax alone cannot distinguish an
authored external custom-type import from a framework import. Document this
limitation on the collector and at its fixture callers; future fixtures that add
external custom types must supply explicit provenance or remain outside this
check. Do not infer provenance from an arbitrary alias spelling or assume that
all external imports are fixed for every valid Loom design.

The replacement checks one Go source file's lexical import boundary, with no
concurrent or protocol state; no new formal model is needed. The implementation
must retain existing shadow-name coverage and must not replace it with weaker
substring matching. Proposed source scope: owning HTTP reservation test,
JSON-RPC name test, a shared internal testing helper and its tests, this plan;
production changes require separately demonstrated evidence and review.

Evidence: `/tmp/loom-import-alias-owner-red.log` reproduces the named assertion;
`/tmp/loom-import-alias-owner-issue.md` is the outgoing draft. Shared #850 captures
show the standalone generated diagnostic file imports only loomdiagnostics.

- [x] Reproduce failure and identify distinct generated import boundaries.
- [x] Independent report/readiness review and filing (#852).
- [x] Replace assertion owner and retain true-collision controls.
- [ ] Independent exact-diff review and atomic integration.

Implementation: `testingx.ServicePackageImportNames` parses rendered import
declarations and reuses the service/views boundary. The HTTP test now compares
observed candidates with the existing reservation owner; its former source-file
scanner is deleted. JSON-RPC adds observed fixed aliases to its existing sorted,
deduplicated compilation sweep, retaining every original local-name case. The
collector and both callers state the controlled-fixture provenance assumption.
Direct controls distinguish same-file imports from standalone and same-package
separate files, exercise views and exact module-prefix boundaries, exclude blank
and dot imports, and verify rejection of a missing reservation.

Validation: focused race-enabled tests pass (testingx 1.404s, HTTP 107.088s,
JSON-RPC 86.108s), including the repaired CI assertion, existing shadow-name
controls, HTTP collision compilation, and expanded JSON-RPC collision compilation.
Affected lint reports zero issues. JSON-RPC now exercises 20 additional HTTP/SSE
alias services and 17 additional WebSocket alias services. Generation captures
compare the unchanged single-service seeds and old/new sweep inputs in separate
processes: all original per-service artifacts retain identical bytes; 420 and
221 new artifacts belong only to added alias services. Six aggregate server/CLI
files change to wire the extra services, plus two recorded name lists. No original
artifact is removed. All 1,473 candidate capture files match the repeat. The
capture includes go.mod and name lists; these counts are not all generated Go
source files. Production generators and the reservation list are unchanged.

Evidence: `/tmp/loom-852-final-tests.log`, `/tmp/loom-852-lint.log`,
`/tmp/loom-852-artifact-hashes.json` and
`/tmp/loom-852-comparison-summary.json`. Comparison generation uses the inspected
22d parent plus the approved #850 test assembly correction; fixture input changes
are solely the recorded additional alias names. The intended integration parent
is `a25383ce104a82257640fa8fc063af06ec9b243b`; both modified test owners are unchanged between those
parents. Final review and integration remain pending.
