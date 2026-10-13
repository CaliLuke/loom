# Decoded body assertion alignment — #843

Parent: `7ff5b71ed4589cd036e2b4e54f4005f36e65eebf` (pushed main).
Scope: test assertions/expected helper fragments only; preserve delivered #838
CLI JSON decode representation and #835 absence-only defaults.

1. [x] Reproduce the six reported CI failures on this parent. Inspect actual
   named helper ownership and existing mounted validation controls before edits.
2. [x] Prepare and independently review a self-contained internal issue report.
   Separate any production defect rather than blessing output as a new golden.
3. [x] Replace stale decoder/type assertions and select transform helpers by
   named responsibility rather than positional offsets. Retain both decoded
   CLI-to-service and service-to-encoder coverage and all prior validation facts.
4. [x] Run focused structural tests and existing bounded CLI/optional-body
   runtime matrices. Compare generated affected designs byte-for-byte at parent,
   candidate and isolated repeat; compile/vet representative modules.
5. [x] Freeze exact atomic diff/evidence for independent review. No production,
   DSL, schema, omission-policy, main checkout or shared guide changes.

No new formal model is needed for test-only alignment of delivered owners;
reuse their existing proofs and behavioral matrices. This is not a full corpus
or full HTTP/gRPC package acceptance run.

## Frozen final result

Issue: https://github.com/CaliLuke/loom/issues/843. The exact reviewed report
was published with bug/code-review/severity:medium. Seven current-parent failures
are recorded in `/tmp/loom-decoded-body-parent-red.log` and
`/tmp/loom-decoded-body-server-parent.log`; the original CI log is
`/tmp/loom-ci-477e86-ci.log`. The extra server case follows delivered #835.

The two optional CLI assertions now identify the non-null decoding helper and
detached CLI type. Transform tests select exactly one section by role and
function name from the correct owner file: CLI input helpers from `cli.go`,
service-to-encoder helpers from `encode_decode.go`, and server helpers from the
server encoder/decoder file. They no longer accidentally compare a response
handler when helper counts/order change. Both directions for both fixtures remain
covered. Four expected helper fragments change; the recursive encoding fragment
is unchanged. Required admission precedes conversion, optional absent B still
receives its default, native values keep their supplied values, and recursive
calls use the decoding carrier's helper.

No generator implementation, runtime, DSL fixture or schema source changes.
The artifact test reuses the existing inner-default, recursive and optional-body
DSLs. It pins the pushed `063198a2f427` module version and explicitly replaces it
with this worktree; Go1.27.2 / GOEXPERIMENT=jsonv2. The pinned module is a
reproducible dependency reference, not the source used for local generation.

## Acceptance receipt

- Direct selected optional-body and both transform-helper suites pass0.666s:
  `/tmp/loom-843-direct.log`.
- Unmodified behavioral matrix passes before assertion edits: optional body
  runtime/CLI/required/null/malformed paths including generated race checks,
  5.571s (`/tmp/loom-decoded-body-parent-wire.log`); shared CLI body/required and
  default-application mounted HTTP/JSON-RPC controls,12.661s
  (`/tmp/loom-decoded-body-parent-cli.log`). Production and every input to these
  behavioral tests remain byte-identical, so this evidence remains valid.
- Three designs generated independently at parent, candidate and repeat:
  `/tmp/loom-843-comparison/{parent,candidate,repeat}`. All36 artifacts per run
  (108 physical artifacts) are byte-identical; no intended output differences.
  Exact inventories are `hashes.json` in that directory. Generation logs are
  `/tmp/loom-843-{parent,candidate,repeat}-artifacts.log`.
- All six parent/candidate modules build and vet; logs
  `/tmp/loom-843-parent-build.log`2.476s and
  `/tmp/loom-843-candidate-build.log`2.738s. Recursive fixture comparison proves
  helper/type closure; existing finite recursive CLI controls prove runtime
  behavior without pretending the fixture's required self-cycle has a finite
  non-null service value.
- Pinned golangci-lint v2.14.0 on `./http/codegen ./http/codegen/testdata` reports
  zero issues (`/tmp/loom-843-lint.log`); `git diff --check` passes.

The parent generation happened before the assertion/golden edits in this same
isolated worktree; only its new inert artifact probe/plan were present. Thus it
used exactly the parent production/design source. Candidate and repeat are
separate Go test processes. No full package/corpus run or new model was used.

Final exact five-source-file patch and evidence are frozen in
`/tmp/loom-843-final.patch` and `/tmp/loom-843-final-manifest.json` for fresh
independent review. No commit/push authorization is claimed by this receipt.
