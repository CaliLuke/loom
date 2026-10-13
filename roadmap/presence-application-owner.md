# Conversion default admission — #835 and remaining M7 sequence

Issue: https://github.com/CaliLuke/loom/issues/835
Owner: `/root/basic_cli_defaults`. Candidate parent:
`0d967158927e65dfc0274da97c7308a677a36c6e` (includes delivered #838 and #839).
Implementation and bounded acceptance are complete; independent final review,
root integration, atomic commit/push and closure remain pending.

## Accepted contract and owner

Defaults fill actual absence, never supplied zero or empty values. A native Go
scalar has no absence bit: its zero is supplied even when a struct literal did
not spell that field. Nil-capable physical storage and absent Optional/Nullable
can trigger defaults; explicit nullable null is supplied. Required admission
observes original presence before default conversion. Selected-view pointer
targets retain absence until their existing materialization boundary.

`TransformAttrs.DefaultForField` resolves one selected target default, source
physical absence capability and target materialization decision. Shared Go,
Optional/Nullable and protobuf conversions consume that record. Physical metadata
precedes semantic kinds; opaque native custom values do not acquire an invented
zero test. The two old native-zero fallback paths and protobuf's separate default
classification are removed. Protobuf bytes now use their actual nil absence.
Default literal construction, type layouts, tags, schema/field identity and
selected-view validation remain with their existing owners.

## Sequential plan

1. [x] Recover prior M7 witnesses and separate native default admission, JSON
   omission, ordinary protobuf collection presence and #740 conversion admission.
2. [x] Reproduce native scalar/alias and protobuf bytes defects, retain real
   absence/custom/required/view/CLI controls, and model current/partial/candidate
   ownership before production.
3. [x] Independent readiness approved; publish the reviewed #835 report.
4. [x] Implement the shared record and remove duplicated zero inference.
5. [x] Discover, separately model/report/review/deliver #838 CLI decoding role;
   constructor input must retain member absence before this owner can consume it.
   #839 separately aligns four unrelated protobuf collection goldens.
6. [x] Advance candidate to delivered #838, preserving every candidate byte;
   validate actual CLI constructor→wire and all bounded consumers.
7. [x] Compare four designs at parent/candidate/independent-process repeat,
   compile/vet, update only intended goldens and document migration.
8. [ ] Fresh exact final review, root integration/atomic delivery, then compact
   temporary artifacts after confirming no active consumer needs them.

## Readiness and prerequisite provenance

Original readiness parent was27255d0258369d8c1604f777ce7a8d600e7dcfd6. The exact
manifest `/tmp/loom-default-application-readiness-manifest.json` has SHA256
`e806736fcfaff1ea4ab9ca4c20480c0fee786b13e0fef0007bf1ecd2597f7ea6`.
Independent review reproduced six native scalar/alias reds and protobuf bytes
absence reds, verified pointer/carrier controls and checked the280-state model.
The old native-string compiled byte-test oracle was intentionally corrected to
preserve supplied empty strings; bytes/custom/pointer/wrapper checks remain.

The candidate exposed a separate CLI producer defect: JSON decoded into encoder
storage had already erased absence. #838 delivered0d9671589 now owns a detached
JSON decoding representation, admission and conversion. Its canonical plan/model
and tests are committed under `roadmap/cli-json-decoding-role.md` and
`http/codegen/tla/cli_json_input`. No duplicate CLI model or implementation is
included here. #838's actual constructor/required/null/custom/recursive/empty
collection tests are reused. A prior four-file #835 overlay composed successfully
with #838; the current checks use the delivered owner directly.

Fast-forward recovery retained every candidate byte; receipt
`/tmp/loom-835-pre-resume-manifest.json`. Only the duplicate helper extraction
needed local reversal before the delivered commit restored it byte-identically.
The three existing production files had no intervening modifications.

## Final acceptance and intended differences

Toolchain: Go1.27.2 with GOEXPERIMENT=jsonv2; golangci-lint2.14.0. Temp modules use
pushed063198a2f427 and explicit local-source replacement for their parent/candidate.

- Parent current-source wire reproduction:
  `/tmp/loom-835-resumed-parent-wire-red.log` fails3.738s on HTTP/JSON-RPC native
  and CLI supplied zeros and response encoding; absent defaults and fully
  populated controls pass. `/tmp/loom-835-resumed-parent-red.log` also records
  protobuf absent-byte request/response failures with empty/populated controls.
- Shared/source-carrier/target-timing/protobuf direct matrix, Go transform goldens,
  compiled bytes/aliases/custom metadata/Optional/Nullable and actual protobuf
  serialization: `/tmp/loom-835-resumed-direct.log`, codegen2.482s and grpc6.139s.
  Nineteen physical-carrier cases across four target modes include required,
  no-default and target-selected occurrence controls.
- Generated mounted HTTP/JSON-RPC native and CLI constructor→wire cases check
  absent defaults, explicit false/0/empty defaulted string and populated values,
  service invocation and request/response payload. Raw required rejection checks
  HTTP400/missing_field/detail and JSON-RPC-32602/message plus no service call;
  Nullable absent/null/empty remain distinct. Final race run7.001s:
  `/tmp/loom-835-resumed-mounted-race.log`. Build and vet run in that same harness.
- Delivered #838 body constructors, scalar CLI defaults/zeros, actual gRPC scalar
  unary/stream/default/required controls and selected-view unary/WebSocket matrix:
  `/tmp/loom-835-resumed-consumers.log` passes HTTP7.377s/generator14.447s.
  Constructor empty collections stay nonnil; their later JSON omission is a
  separate M7 boundary, not claimed repaired by this default-admission change.
- Four bounded designs (`transports`, `bytes`, `cli`, `views`) produce85artifacts
  per run. `/tmp/loom-835-delivered-comparison/hashes.json` verifies all255files;
  candidate/repeat are identical. Nine intended differences in `differences.json`
  and `output.patch`: three protobuf bytes constructors add nil-default branches;
  HTTP/JSON-RPC client/server transforms remove native-zero fallback; CLI/server
  required-member conversion removes unreachable default-after-missing branches.
  Schemas, declarations, tags and the full scalar-CLI/view designs are unchanged.
  Final populated test controls changed only harness expectations/inputs, not the
  design functions used for these artifact comparisons.
- Four parent modules build/vet4.697s; four candidate modules5.142s:
  `/tmp/loom-835-delivered-build-{parent,candidate}.log`. The identical repeat
  does not repeat build/vet. Generation uses matching generate logs.
- Ten checked-in golden updates have the same scope: nine shared transform
  snippets lose native-zero replacement; one protobuf snippet uses source bytes
  nil instead of an invalid slice-zero comparison. No unrelated golden alignment.
- Scoped generator/protobuf lint and explicit nested fixture-package lint both
  report0issues: `/tmp/loom-835-delivered-{lint,fixture-lint}.log`.
- Fresh model checks `/tmp/loom-835-final-model-*.log`: zero and target-only fail
  PreserveSupplied at147states, bytes-kind fails FillAbsence at208, nil-only fails
  FillAbsence at236, carrier passes280. Models assume required input admission;
  tests establish concrete representation/validation and generated execution.

Reproduce with GOEXPERIMENT=jsonv2 and LOOM_DIR pointing to the worktree:
`go test ./codegen ./grpc/codegen -run '^Test(GoTransform|TransformDefaultApplication|TransformDefaultUsesTargetOccurrence|DefaultApplicationScalarValues|DefaultApplicationCarrierControls|BytesDefaultTransformLayouts|DefaultValueStorageMetadata|ProtoDefaultApplication|ProtoDefaultBytesGenerated|ProtoBufTransform|ProtobufScalarPresence|ProtobufScalarAliasLowering)$' -count=1`.
For consumers: `go test ./http/codegen ./codegen/generator -run
'^Test(ResultViewPresenceGeneratedIntegration|DefaultApplicationGenerated|DefaultApplicationCLI|CLIJSONBodyContracts|GRPCScalarPresence)$' -count=1`.
Run the mounted default harness with GOFLAGS=-race. For artifacts, set
LOOM_DEFAULT_APPLICATION_OUTPUT and run TestDefaultApplicationArtifacts, followed
by TestDefaultApplicationArtifactBuilds; repeat generation in a new process.

The earlier full grpc/codegen sweep timed out600s in an unrelated wrapper-contract
matrix and is not a green gate. It is not repeated; bounded affected evidence above
is the ticket acceptance. The earlier four stale collection goldens were #839 and
are delivered, not part of #835.

## Remaining M7 work, explicitly outside this atomic diff

1. Reconcile JSON runtime tags and retained value projection around physical
   presence. Empty omission remains separate; full authored JSON tags and custom
   codecs must retain their contracts. No isolated omitempty replacement.
2. Ordinary protobuf collection message presence needs its own wire/schema owner.
   #684 repaired wrapper contents, not ordinary repeated/map field presence.
3. Repeated text-array encodings still collapse some empty/absent states; do not
   silently replace authored formats or re-ask settled explicit-zero semantics.
4. #740 inline protobuf nil object conversion is independent admission ownership.
   The saved current-parent generated probe has six panics/four controls, with
   recursive helper versus inline conversion and binary serialization relevant.
   No nil-element policy or local pointer guard is inferred here.

The protected worktree retains broader readiness-only inputs under
`roadmap/presence-investigation/` and `codegen/tla/explicit_presence/`; these are
excluded from the #835 patch/manifest. Original broad readiness remains in the
explicit-presence-contract worktree. The #740 log is
`/tmp/loom-presence-application-740-red.log`; historical CLI producer input is
`roadmap/presence-investigation/default_application_cli_body_test.go.txt`.
Completed #684 cleanup receipt remains `/tmp/loom-684-cleanup-receipt.json`;
managed source worktrees were protected and not manually removed.
