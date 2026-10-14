# Physical-route fixture adoption — #902

Parent: `2de11550d`. The shared physical-route contract delivered under #781
correctly rejects unrelated alias and example fixtures that reused a route.
This repair changes test inputs and expectations only; production admission,
service/package names, alias competitors and multipart settings stay intact.
The separate header formatter expectation repair is #903.

1. [x] Reproduce current alias, example, scaffold and static-file admission
   failures before changing fixtures. Give unrelated HTTP services distinct
   routes and unrelated JSON-RPC dispatchers distinct physical paths.
2. [x] Give the excluded endpoint and excluded service in `WithTagsDSL` distinct
   routes. `openapi:generate=false` suppresses documentation, not runtime
   admission. Preserve the tag and inclusion/exclusion assertions and golden.
3. [x] Keep the static-file/endpoint collision intentionally invalid and assert
   its rejection at admission, including both owners. Keep the recursive parent
   invalid; update only the exported-design diagnostic expectation. The direct
   expr cycle assertion was already correct.
4. [x] Regenerate the static server golden through its owning test. The only
   golden delta is three direct `mux.Handle` registrations replacing helper calls
   after the delivered mount-owner repair; preserve field-doc assertions.
5. [x] Run affected direct checks, existing alias/aggregate module compilation,
   the two exact exported cases and three multi-service example CLI builds.
   Compare bounded actual output and separate-process repeats.
6. [ ] Independently review the exact final diff, deliver one atomic commit, then
   preserve concise evidence and remove inactive temporary outputs.

## Evidence and limits

`/tmp/loom-902-parent.log` reproduces the direct failures. The archived parent
also rejects `WithTagsDSL`; the ordinary `ServerFileServerDSL` builds and vets.
Corrected direct checks pass across example, gRPC, HTTP, OpenAPI and JSON-RPC
packages. Existing alias and aggregate module tests pass. The corrected exported
selection executes both `WithTagsDSL` and the intentionally invalid recursive
parent; the latter verifies its DSL-stage diagnostic rather than expecting
compilation. All three multi-service examples run actual CLI generation/example,
module tidy, build and vet using the existing exported-design harness with an
explicit three-design catalog addition in a test-only capture overlay.

Nine bounded generated modules produce 332 artifacts. A fresh-process repeat
produces identical bytes; redundant builds of identical repeat output are
skipped. All 18 ordinary static-module artifacts match the archived parent.
Rejected parent specimens have no fabricated generated baseline: their recorded
admission failures are the before evidence. Newly admitted output reflects only
the authored route changes, with alias/naming assertions retained. The WithTags
OpenAPI golden stays unchanged. Scoped configured lint reports zero issues.

`/tmp/loom-902-final-manifest.json` records exact sources, inputs, output ledgers
and passing gate logs. Initial selectors that ran no tests are excluded from
acceptance; corrected logs list the executed subtests. A temporary parent-capture
path issue was corrected before the recorded 18-artifact comparison. No full
corpus, new model, production workaround or aggregate CI-green claim is made.
