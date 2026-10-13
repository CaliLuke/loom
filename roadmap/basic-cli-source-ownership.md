# Basic CLI source ownership — #842

Parent: pushed `b963d8685`. Independent owner and acceptance review approved
the readiness plan before production edits. Preserve the existing Basic-default and credential-validation
contracts; repair duplicate CLI input ownership without rejecting valid mappings.

## Investigation sequence

1. Reproduce the preserved required Basic query/header witness on this parent.
   Inspect `buildPayloadFieldArgs`, `buildBasicAuthCLIArgs`, `buildFlags`,
   `cliPayloadInit` and both HTTP/JSON-RPC callers, using previous #794/#838 plans.
2. Compare direct source identities, physical argument types, field bindings,
   requiredness/defaults and generated declarations for Basic-only, ordinary
   mappings, custom Go field names, optional/defaulted credentials, repeated
   non-body mappings, scalar Body and aggregate body duplication.
3. Identify the smallest shared owner that selects one direct CLI input per
   payload source. Preserve distinct body aggregate inputs and any existing
   documented precedence until concrete evidence settles that separate boundary.
   Do not deduplicate by printed Go names or patch transport templates separately.
4. Model argument/source selection and assignment completeness before production.
   Reproduce current duplicate ownership, compare candidate source-aware selection,
   and document model assumptions and the Go/generated acceptance seams.
5. Freeze readiness evidence, proposed implementation loci and bounded acceptance
   matrix for independent review. No generator implementation before that gate.

## Known evidence and constraints

The preserved minimal design under `/tmp/loom-718-basic-cli-parent/design` maps
Basic username `login` to query `signin` and password `passcode` to header
`X-Proof`. Parent generation succeeds; CLI compilation reports duplicate login/
passcode formal parameters and locals. The generated payload repeats assignments.
The old generated module's local source replacement was an intentionally removed
comparison copy; new probes must use this worktree explicitly, not that stale path.

`buildPayloadFieldArgs` retains logical names in AttributeData.Name, separately
from allocated VarName and physical FieldName. `buildBasicAuthCLIArgs` emits the
same logical username/password again. `buildFlags` concatenates ClientArgs and
CLIArgs; makeFlags creates flags, locals and assignments for every entry. JSON-RPC
uses the same ClientCLIFilesForTransport implementation. Its runtime wire bindings
already reuse service fields, so the duplicate is in CLI input construction.

The #794 repair preserves effective defaults on Basic argument records; it must
not be undone. #838 gives JSON body input a detached decoding representation;
its aggregate body carrier and conversion must remain intact. Public CLI argument
spelling, named/custom field conversion, optional/defaulted representation and
validation need explicit controls, not a final-name set filter.

## Readiness findings and proposed owner

- [x] Reproduced on pushed parent b963d8685, with no production changes.
- [x] Identified the common CLI projection boundary and both transport consumers.
- [x] Compared explicit and implicit source metadata and allocated local names.
- [x] Modeled append, printed-name selection and logical-source selection.
- [x] Independent readiness review, then implementation.
- [x] Direct regression, generated runtime controls and bounded byte comparison.
- [ ] Independent final review; orchestrator integrates one atomic #842 commit.

The private `cliPayloadArgs` owner beside `buildFlags` projects the
existing ClientArgs followed by CLIArgs into a fresh slice. For direct payload
fields (nonempty FieldName), retain the first argument per logical Name. Keep
body/whole-payload carrier arguments unchanged. Both flag construction and
payload initialization must consume this same selected sequence. Retain complete
existing argument records: representation, effective default, requiredness,
validation, custom service field binding and named conversion. Do not deduplicate
printed names, alter transport constructor arguments, prune wire mappings, or
mutate cached slices. This also coalesces repeated ordinary mappings of the same
source, which exhibit the same independent-input ownership problem.

The selected ordinary argument already derives its type and validation from the
service payload occurrence through `buildTransportAttributeData`; Basic arguments
are used when no ordinary direct operation provides that source. Existing source
metadata agrees for the required, optional and defaulted fixtures. No new policy
for empty flags, body inputs or authentication is introduced. Basic flags still
follow body transformation in generated payload assignment; conflicting body and
direct values retain the existing order rather than becoming a new source merge.

The `BasicCLISourcesDSL` uses named constrained credentials and custom service
field names. Current direct checks fail for query/header, optional/defaulted,
path/cookie and repeated mappings. Generated full CLI modules fail for those same
cases in HTTP and JSON-RPC (path/cookie HTTP only). Basic-only, scalar-body with
ordinary field spelling and aggregate-body controls all build and vet in both
transports. Logs: `/tmp/loom-842-direct-red.log`,
`/tmp/loom-842-generated-red.log`, `/tmp/loom-842-controls.log`.

The first scalar-body control independently exposed a separate defect: selecting
`Body("login")` with custom Go field AuthUser emits nonexistent Login selectors
in server construction, client body conversion and CLI. An overlay removing Basic
security still fails both transports. That independently reproduced defect is tracked as #847; it is
not fixed or hidden by #842. The ordinary scalar-body control is green, and all
non-scalar mapped cases retain custom-field coverage.

`http/codegen/tla/cli_sources` reproduces duplicate source ownership for append
and printed-name selection (exit12,136 states each). Logical source selection
passes all448 states. The bounded model covers all64 active subsets of six
operations, preserving whole-body carriers and coverage of wire sources. It does
not prove Go representation or validation correctness.

## Implementation acceptance and bounded comparison

1. Change the direct regression to consume the actual source-selection owner.
   Assert unique logical direct sources and unchanged representative records,
   flags, metadata and cached constructor arguments. Add a repeated non-Basic
   mapping control to demonstrate the same owner independently of auth.
2. Keep the existing eight HTTP/seven JSON-RPC specimen shapes. Compile and vet
   every full module; check actual CLI parsing/builder payloads and outbound
   Basic plus authored query/header/path/cookie copies. Required missing and
   constrained invalid credentials must still fail. Optional omission and
   effective defaults/explicit overrides must retain their #794 behavior.
3. Include body controls with differing body/direct credential values and assert
   the established final assignment order. Preserve #838 JSON decoding presence
   and conversion; do not replace a body carrier with a direct scalar argument.
4. Generate parent/candidate/fresh-process candidate from the same bounded
   specimens. Only CLI declarations, flags/help and duplicate payload assignments
   associated with repeated direct inputs may change; service, wire encoders,
   server constructors and OpenAPI bytes must match. Candidate runs must match
   byte for byte. Parents are expected to fail only the reproduced duplicates;
   scalar custom-field residual is outside this acceptance matrix.
5. Run affected Basic/CLI suites, targeted lint and model correspondence review.
   Use existing valid evidence for unchanged #794/#838 surfaces; no full exported
   design corpus is a local acceptance gate.

The final exact readiness fixture rerun is recorded in
`/tmp/loom-842-readiness-tests.log`: all five HTTP/four JSON-RPC mapped cases
fail the reproduced duplicate declarations; Basic, scalar-body and aggregate-body
controls build/vet successfully. Direct repeated-input checks fail as expected.
The independent #721 Basic path witness is also preserved at
`/tmp/loom-721-basic-path-repro.go` and
`/tmp/loom-721-basic-parent-build.log` (source0f1fc3dc395).

Readiness evidence is frozen separately from implementation. The original readiness regression files failed before the approved correction;
the implemented suite now passes.

## Final implementation and validation

`cliPayloadArgs` is the only production owner added. `buildFlags` calls it
instead of appending the cached ClientArgs and CLIArgs slices. This removes
multiple declarations, loads and writes for a single direct payload source while
retaining the first complete record. Both command-level flag emission and the
service payload builder consume this sequence. There are no transport-specific
patches, new exported APIs, compatibility paths, server edits or wire edits.

The fixture now includes ordinary repeated mappings without Basic security,
showing that source ownership applies to the same cause independently of auth.
Direct tests retain pointer-identical complete records, independent selection
slice storage and unchanged cached argument sequences. Generated tests use the
actual CLI parser, inspect named payload fields and capture outbound requests.
They assert every query/header/path/cookie copy, Basic values/presence, defaults,
optional absence, missing-required rejection and both invalid constrained
components. Scalar and aggregate body inputs intentionally contain different
values; the direct flags retain the prior final value in both payload and wire.

Validation uses Go1.27.2, `GOEXPERIMENT=jsonv2`, explicit local `LOOM_DIR`, and the
root's pinned dependencies. The generated temp modules require the existing
verified pushed version `v1.10.0-alpha.9.0.20261013020843-063198a2f427` and replace
it with the selected source for local validation.

- The complete new runtime matrix passes: nine HTTP and eight JSON-RPC modules,
  each built/vetted with four generated tests run under `-race` (68 cases total).
  The initial focused run took31.023s/27.453s. The final affected-suite invocation
  includes these same generated race tests plus Basic defaults/presence/validation,
  existing ClientCLI goldens, JSON CLI and field-loading controls: HTTP41.546s,
  JSON-RPC37.490s and shared CLI0.495s, all passing.
- Pinned golangci-lint reports zero issues for affected codegen packages and the
  shared service testdata. Repository gofmt, filesize, NameScope and JSON-v2
  gates pass; existing filesize warnings are unchanged.
- The model source/configurations remain identical to readiness: current append
  and printed-name selection each fail UniqueDirectSource at136 states; logical
  source selection passes448. The implementation correspondence README now names
  the actual owner. These are bounded checks, not a proof of Go code generation.

The bounded output comparison uses an immutable archive of parent b963d8685 with
only identical new fixture/test inputs added (and an empty local Git repository
for the existing source resolver). The parent artifact-only test excludes the
candidate-only direct helper test. Each candidate repeat is a separate Go test
process. Temporary setup attempts without Git metadata were discarded; accepted
parent evidence uses the fresh `parent-final` output directory and successful
artifact-generation log.

| Source | Generated artifacts | Changed from parent | Fresh-process differences |
| --- | ---: | ---: | ---: |
| Parent | 222 | — | — |
| Candidate | 222 | 22 | 0 |

Every specimen has complete service, client, server and CLI Go artifacts; HTTP
specimens additionally include OpenAPI JSON/YAML. Only each affected specimen's
aggregate CLI and service client CLI change: duplicate flags/help/examples,
formal/actual parameters, locals, validation blocks and repeated payload writes
are removed together. Six HTTP and five JSON-RPC specimens change; these include
the ordinary repeated-mapping control. Basic-only and both body controls are
byte-identical. All service, wire, server and OpenAPI artifacts are byte-identical.
No artifact was added or removed. Existing checked-in CLI goldens pass unchanged.

All17 parent modules were built: nine reproduce duplicate declarations as
expected; eight controls build and vet. The ordinary repeated-mapping parents
compile but expose two direct flags for one source; their corrected CLI now
accepts one logical input. Candidate modules all build, vet and pass their
runtime race checks. Fresh byte-identical output is not rebuilt redundantly.

Compact evidence: `/tmp/loom-842-artifact-hashes.json`,
`/tmp/loom-842-generated.diff`, `/tmp/loom-842-parent-builds.json`,
`/tmp/loom-842-affected.log`, `/tmp/loom-842-lint.log`, and the original model
results. Preserve reproduction inputs and hash summaries before removing the
inactive generated trees and checker state after independent final review.
