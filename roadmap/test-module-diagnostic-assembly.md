# Generated test-module diagnostic assembly — #850

Parent: `22d000aa8ce70a8f5d1fe72354d0963be240e29a`.
Scope: manual test-module assembly only; the full generator already emits the
required metadata, and no production behavior change is proposed.

1. Reproduce HTTP and JSON-RPC local-name sweep failures. Trace example diagnostic
   imports to the metadata files the full generator assembles. Record the same
   omission in the mixed HTTP/JSON-RPC test helper separately from observed CI.
2. After independent report/readiness review and filing, append DiagnosticFiles
   in renderHTTPModule, renderJSONRPCModule and both transport portions of
   renderMixedHTTPJSONRPCModule. Use each helper's existing ServicesData owner;
   do not construct per-caller or per-service exceptions.
3. Remove the redundant explicit DiagnosticFiles rendering from the existing
   TestGeneratedCredentialDiagnostics caller now covered by renderHTTPModule.
   Search for other compensating callers before editing; do not duplicate files.
4. Add direct assembly controls for HTTP, JSON-RPC and mixed credential-bearing
   designs plus no-security controls. Assert expected metadata packages are
   present exactly once and inactive services get none; append normal example
   server/CLI files and compile/vet the complete modules. Use valid JWT/token
   mappings to isolate assembly from the separate Basic CLI and selected-body
   defects. Retain every existing local-name sweep assertion unchanged.
5. Run the actual failing HTTP/JSON-RPC sweeps, existing credential diagnostic
   runtime checks, mixed example control and the new direct assembly controls.
   Compare a bounded generated selection at parent/candidate: only the missing
   diagnostics.go artifacts may be added by manual helpers; all preexisting
   files must match byte for byte. Candidate fresh-process output must match.
   Normal full-generator output remains unchanged by construction because the
   patch touches test files only. No exhaustive exported-design audit is needed.
6. Run affected lint, independent exact-diff review and root atomic integration
   as a separate ticket from inline-body assertions. Preserve compact evidence
   before temporary artifact cleanup.

Ownership evidence: `codegen/generator/transport.go` already calls
httpcodegen.DiagnosticFiles for both transport roots. The three test assemblers
omit those calls; generated ExampleServerFiles/ExampleCLIFiles import the missing
per-service transport package when credentials exist. The current HTTP diagnostic
integration test compensates with its own explicit rendering. Root and independent
logs are `/tmp/loom-ci-22d-regressions.log` and
`/tmp/loom-ci-owner-readiness-red.log`. The mixed helper is an inspected shared
cause, not a claimed observed CI failure.

No new state/protocol model is warranted: the production diagnostic contract and
its existing model remain unchanged. This repair aligns linear test source
assembly with the established full-generator owner.

Proposed source scope: two helper files, removal of the redundant HTTP diagnostic
caller, focused HTTP/JSON-RPC test files and this plan. The report draft is
`/tmp/loom-render-diagnostics-issue.md`.

- [x] Trace full-generator owner and missing test-assembly calls.
- [x] Reproduce local-name sweep failures on current parent.
- [x] Independent report/readiness review and filing (#850).
- [x] Shared test assembly repair and bounded controls.
- [ ] Independent final review and atomic integration.

Implementation and validation: the HTTP assembler, JSON-RPC assembler and both
parts of the mixed assembler now append the established DiagnosticFiles owner.
The previous HTTP diagnostic-test workaround is removed. New direct controls
cover active JWT, inherited security disabled at the method, and ordinary fields
for HTTP, JSON-RPC and mixed modules. They first fail with missing metadata on
the parent helpers, then verify exactly the expected service packages and render
complete service/server/CLI examples for full module build and vet.

The selected outer race-enabled test run passes (HTTP 109.153s, JSON-RPC 50.130s):
new assembly controls, both existing local-name compilation sweeps, the existing
credential diagnostic runtime test (itself runs generated code with race), and
the existing mixed example compilation control. Affected lint reports zero
issues. Only whitespace formatting of empty DSL functions followed these checks;
the parsed fixture and generated behavior are unchanged.

Three independent capture processes compare the manual assemblers' complete gen
trees: 118 existing artifacts retain identical bytes; only four expected
secured-service diagnostics.go files are added (HTTP, JSON-RPC, both mixed
transports). All 122 candidate artifacts match the repeat. Capture ends before
example rendering; separate module tests compile and vet the examples. No
production source, reservation policy, or existing local-name assertions change.

Evidence: `/tmp/loom-850-assembly-red.log`, `/tmp/loom-850-final-tests.log`,
`/tmp/loom-850-lint.log` and `/tmp/loom-850-artifact-hashes.json`. The source-level
import-reservation heuristic failure is separately reproduced and remains outside
this ticket; this is not a claim that the entire framework suite is green.
Final review and integration remain pending.
