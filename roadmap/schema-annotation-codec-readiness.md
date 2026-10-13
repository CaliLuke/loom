# Schema annotation codec boundary — #818

Base: `b756c06c3b71d20411fdd6252c8cb158cf3884df`.

1. [x] Trace raw mapped SSE and custom text-cookie defaults to
   `expr.ValueContext.schemaAnnotationValue`. The current branch confuses a
   supported authored value with a supported target JSON codec.
2. [x] Reproduce with direct default/enum controls, actual cookie generation,
   and mapped SSE generation. Keep invalid builtin numeric and sibling controls.
3. [x] Model source validity independently from target-codec support. Preserve
   the old false rejection and a blanket-fallback counterexample.
4. [x] Obtain independent readiness review before production edits.
5. [x] Retain checked declared annotations with an explicit Unsupported
   projection at opaque target boundaries; preserve existing builtin rejection.
6. [x] Run affected tests, bounded parent/current/repeat generation, compile,
   and affected lint. Freeze an exact manifest for independent final review.

The accepted value-contract design preserves non-JSON locations and custom
codec ownership. A retained declared value is not the custom codec's JSON output
and must never become a builtin membership claim. The shared annotation owner
already retains opaque source leaves and checks known builtin siblings. Extend
that owner to valid builtin authored values at opaque target boundaries without
transport-specific exceptions. Source-invalid, absent, or unretained values
remain rejected; shortest-decimal numeric membership still applies to builtin
JSON subtrees. No DSL or runtime codec policy changes.

Acceptance is bounded to root text/raw/custom and nested custom default/enum
annotations, known builtin invalid siblings, invalid source values, existing
binary/exact-decimal controls, the existing UUID cookie fixture and mapped SSE
fixture. Unaffected JSON annotations and generated Go remain byte-identical.
No full exported-design sweep is needed. #727 is frozen separately and waits
for this prerequisite; its selected-view production change is outside scope.

Readiness evidence (Go 1.27.2, TLC 2.19 / tools 1.7.4):

- `go test ./expr -run '^TestSchemaAnnotations' -count=1` reproduces seven
  intended positive failures; JSON, invalid-source, existing exact-numeric and
  opaque-source sibling controls pass. `/tmp/loom-annotation-red.log`.
- The focused existing cookie CLI case fails during annotation preparation:
  `/tmp/loom-annotation-cookie-red.log` (16.208s).
- `SSEMappedAnnotationDSL` fails at the same owner:
  `/tmp/loom-annotation-sse-red.log` (6.200s).
- Model Old fails PreservePortable (32/32 states); Blanket fails
  RejectKnownFailure (31/31); Candidate passes (48/48).
  Logs `/tmp/loom-annotation-model-{Old,Blanket,Candidate}.log`.

Proposed production delta is restricted to `schemaAnnotationValue`: remove the
source-Unsupported prerequisite when projection is Unsupported and the declared
value is retained; retain `checkKnownSchemaAnnotation` without duplication.
Any discovery that this cannot preserve the existing boundary requires updating
this readiness record before expanding scope. No production file is edited at
readiness freeze.

## Implementation and acceptance

Root independently approved the exact readiness manifest before production
editing. The implementation removes only the source-outcome prerequisite in
`schemaAnnotationValue`; retained ownership, projection errors, known builtin
checks and Unsupported outcome are unchanged. No transport has its own fallback.

- Full `expr`, HTTP `internal/representation`, and OpenAPI `internal/ir` packages
  pass: `/tmp/loom-818-packages.log` (1.528s / 0.250s / 7.985s).
- The two rendered annotation regressions fail on the exact parent and pass
  after correction. JSON/YAML parse with libopenapi and pass the pinned Vacuum
  rules: `/tmp/loom-818-openapi-red.log` and `/tmp/loom-818-openapi.log` (green 24.508s).
- Pinned golangci-lint 2.14.0 covers `./expr/...`, explicit
  `./http/codegen/testdata`, and `./http/codegen/openapi/v3`: zero findings in
  `/tmp/loom-818-lint.log` and `/tmp/loom-818-openapi-lint.log`.
- The five-design comparison uses the UUID cookie default, mapped SSE annotation,
  custom cookie without a default, normal mapped SSE, and JSON inline-object
  default. Parent fails only the first two during annotation preparation; all
  five current modules build and vet. Generation runs in separate processes with
  the same CLI basename and module name. Initial compilation used phase-specific
  CLI basenames; subsequent generation changes only that header comment, so
  compilation evidence is reused rather than rebuilding identical code.
  Inputs, script, exact hash inventories and summary are retained under
  `/tmp/loom-818-comparison` until independent review is complete.
- The separately frozen #727 manifest plus this one production prerequisite,
  copied into `/tmp/loom-818-727-overlay`, passes the full
  `TestDesigns/http/codegen/SSEViewedResultDSL` gen/example/tidy/build/vet pipeline
  (27.292s): `/tmp/loom-818-727-overlay.log`. Frozen #727 files were not edited.

Intended difference: the two formerly rejected designs now produce their complete
artifacts with the authored default/enum values retained. Normal controls must
remain byte-identical; custom target JSON is never claimed to have been proved.
No runtime encoding, DSL, public schema metadata or custom-codec policy changes.
Regeneration is sufficient for affected valid designs; no design migration.

The model is bounded evidence about admission logic, not a proof of graph
traversal or generated code. Existing numeric proofs are unchanged because no
numeric semantics changed. No exhaustive corpus was run. Root owns final review,
atomic commit, push and issue closure. Preserve compact evidence, then delete
unused temporary artifacts once independent review completes.

Final comparison: 84 artifacts across five designs are byte-identical in
independent current/repeat generations; all three successful parent controls are
byte-identical to current. Exact summary: `/tmp/loom-818-comparison/summary.json`.

Independent root review approved the final candidate, including the fixture GoDoc
correction, with no unresolved findings. Integrated guidance records the codec
boundary and model index. Root reran direct annotation and rendered OpenAPI
regressions successfully before the final integrated review.

Integrated review identified and corrected a LOW validation-metadata issue:
all 15 comparison modules and the comparison script now require the verified
pushed base `v1.10.0-alpha.9.0.20261013013155-b756c06c3b71`. Explicit local
replacements still select the exact parent/candidate sources under test.
`go mod tidy` passes in every module; source and generated artifacts are unchanged,
so generation/build evidence is reused. Receipt:
`/tmp/loom-818-comparison/pin-receipt.json`.
