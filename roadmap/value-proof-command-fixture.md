# Proof gate fake command inventory — #816

Parent: `0b565742284e923adcc54d80ae670820c7ec5b83`.
Owner: the deterministic fake Lake command contract in
`scripts/value_contract_proof_test.go`. No production script or Lean change.

## Sequence and acceptance

- [x] Reproduce the unchanged parent's 13 failing subtests out of 25.
- [x] Add the numeric admission unexpected-success regression and invocation
  assertion; observe the new regression fail before changing the fake.
- [x] Add only the existing numeric admission command to the fake's explicit
  command allowlist. Preserve unknown-command rejection and existing diagnostics.
- [x] Pass all 26 gate cases, affected scripts tests, vet, and pinned lint.
- [x] Freeze exact diff and evidence for independent review.
- [ ] Root integrates one reviewed atomic commit, pushes, and closes #816.

The production gate already invokes `NegativeNumericValueAdmission.lean` and
requires its false-proposition diagnostics. The fake's existing default negative
case emits those diagnostics. The repair adds the missing command and the same
success/invocation coverage provided for the other negative controls. A formal
model and generated transport outputs are not relevant to this fixture-only
change. It does not establish that actual Lean proofs pass.

## Evidence

Parent reproduction: `go test ./scripts -run '^TestValueContractProofGate$'
-count=1 -v`, `/tmp/loom-proof-fixture-parent.log`, fails in 12.891s. The success
case and twelve later audit/kernel checks all encounter the unrecognized
numeric-admission command. Earlier controls still reach their intended failures.

Red-first added regression: `go test ./scripts -run
'^TestValueContractProofGate/numeric_value_admission_control_succeeds$'
-count=1 -v`, `/tmp/loom-proof-fixture-new-red.log`, fails in 0.977s because the
fake rejects the command before the injected unexpected-success behavior.

Candidate: `go test ./scripts -count=1 -v` passes in 28.055s, including all
26 proof-gate cases; real-Lean opt-in audit tiers remain skipped as designed.
`go vet ./scripts` passes, and pinned `golangci-lint run --config .golangci.yml
./scripts` reports zero issues. Logs are `/tmp/loom-proof-fixture-candidate.log`,
`/tmp/loom-proof-fixture-vet.log`, and `/tmp/loom-proof-fixture-lint.log`.
Toolchain: Go 1.27.2 darwin/arm64, golangci-lint 2.14.0.

The same failure appears in CI run 38328954171, job 115074045636 at `7a61ec0e0`
(value-contract gate), and race job 115074353568 at `285da6b0e`. These are
corroborating CI receipts; the parent and red-first logs above are local evidence.

Final scope is three test-file additions and this receipt. The production script
and all proof sources are byte-identical to the parent. Acceptance is restoration
of the fake command protocol and each intended orchestration assertion, not
real-Lean proof validation. Commit, push, and issue closure remain root-owned.
