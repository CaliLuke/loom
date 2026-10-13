# Constraint domain admission (#695)

Base: `063198a2f`.
Status: implementation and bounded acceptance complete; independent final review pending.

## Contract and owner

Exclusive endpoints at the same exact value define an empty interval. Length
bounds must be nonnegative; zero is valid. Authored constraints apply through
named ancestry and Extend, so checking only local ValidationExpr leaves inherited
contradictions undetected. Current Validate uses > for the exclusive/exclusive
pair and never checks negative lengths. Attribute effective admission currently
runs only for enums/defaults. Extend has its own more-correct interval check.

Proposed repair: one expr domain-admission helper shared by local ValidationExpr
validation and effective finalized constraint construction. Check negative
lengths at each authored layer before tighter-bound selection can erase them;
check the selected exact numeric interval and merged length interval afterward.
Use existing EffectiveConstraints bound selection/provenance. Attribute admission
must run for numeric/length ancestry even without enums/defaults. Remove the
Extend-specific duplicated interval/length checks once the shared owner rejects
them. Keep local same-side minimum/exclusive-minimum declaration restrictions,
enum refinement, exact-number grammar and carrier admission unchanged.

This is endpoint/length contract admission, not a satisfiability solver for
patterns, arbitrary codecs, enum intersections, or discrete representable values.
Redundant bounds outside a native carrier remain accepted; no new range policy.

## Accepted public policy

The user explicitly adopted rejection of contradictory inherited bounds after
reviewing the concrete base `Minimum(10)` plus derived `Maximum(5)` example.
The alternative was retaining the documented admission of inherited empty
intervals; it would disagree with local/Extend admission and produce contracts
with no admissible values. The accepted rule rejects consistently before
generation. Equal inclusive endpoints and zero lengths remain accepted; equal
exclusive endpoints, reversed bounds and negative authored lengths reject.
The public DSL guide replaces its prior admission sentence. Existing affected
designs need corrected bounds before regeneration. This acceptance resolves the
previous policy pause; no outstanding user decision remains for #695.

## Sequential tasks

- [x] Read original issue, local validator, effective owner and Extend checks.
- [x] Create managed isolated worktree and durable plan.
- [x] Add direct original failures and effective/named/inherited/Extend cases.
- [x] Confirm controls for equal inclusive endpoints, proper intervals, zero
  lengths, tighter inherited bounds, exact large integers/decimals and no mutation.
- [x] Model local versus composed domain admission before production.
- [x] Freeze readiness and obtain independent architectural approval.
- [x] Implement shared owner; remove superseded composition checks.
- [x] Run expr/DSL tests, bounded CLI accepted/rejected design checks, and compare
  representative valid numeric/length generated outputs byte-for-byte with base
  and independent repeat; compile affected generated modules. No corpus audit.
- [x] Update public/internal guidance.
- [ ] Obtain independent exact final review.
- [ ] Root integrates one atomic commit, pushes, closes #695 and records cleanup.

## Acceptance and scope

Reject the three original examples and contradictions split across declaration
ancestry or Extend without requiring a default/enum to trigger validation.
Negative authored lengths remain errors when a valid tighter child would mask
them. Admit equal inclusive bounds, proper exclusive intervals and zero lengths.
Preserve exact comparisons beyond float precision, immutable authored constraints,
existing local diagnostic context and source provenance for effective failures.
All still-valid generated output is expected byte-identical; rejected designs
must fail before generation with an actionable domain diagnostic.

## Readiness evidence and bounded final checks

There are 18 red leaf cases: three local ValidationExpr rules, six effective
ancestry cases, and nine DSL cases. The two Extend contradiction controls already
reject correctly and must keep rejecting after their duplicate checks disappear.
Existing effective-constraint and Extend tests plus valid new cases pass in
0.529s/0.167s. Scoped expr/DSL golangci-lint reports zero issues. Logs are
`/tmp/loom-695-readiness-{red,controls,lint}.log`.

The model rejects Local and MergedOnly with intended Admission counterexamples
(437/437 and 682/682 states); Checked passes 1297/865 states. Logs are
`/tmp/loom-695-model-{Local,MergedOnly,Checked}.log`. Sources/configs and their
implementation correspondence remain in `expr/tla/constraint_domain`.

Final accepted-output sample: reuse QueryIntAliasValidateDSL for named integer
bounds and PayloadBodyPrimitiveFieldArrayUserValidateDSL for named array length
and element constraints. Add the valid boundary design from the DSL test as one
bounded module for exact exclusive/inclusive endpoints and zero lengths. Compare
all emitted files at parent/current/independent repeat and build/vet current and
parent modules with a verified pushed pin plus explicit source replacements.
Keep three original and named-ancestry invalid designs in CLI rejection checks;
no output comparison is possible for candidates intentionally rejected before
generation. A whole-corpus audit is not a requirement.

## Final acceptance evidence

One expr owner now checks local ValidationExpr and immutable effective constraint
construction; composition no longer duplicates numeric/length comparisons.
Authored-layer validation precedes folding, preventing a tighter valid bound from
hiding a negative ancestor length. Effective admission runs for numeric/length
ancestry without requiring enums or defaults. Existing enum/default handling and
carrier policy remain unchanged.

Full expr/DSL suites pass (0.655s/0.255s). The codegen package's only initial
failure was `TestEffectiveNumericAliasContradictionRemainsAdmitted`, which encoded
the superseded public policy. Its identical named design now asserts rejection
before generation and effective-owner rejection. All other codegen tests passed
in that run; the affected numeric-alias tests pass (0.481s) after the policy-test
update. Scoped expr/DSL/codegen golangci-lint reports zero issues. Logs:
`/tmp/loom-695-owner-final.log`, `/tmp/loom-695-codegen-policy.log`,
`/tmp/loom-695-final-lint.log`. Earlier full expr/DSL also passed after the owner
change in `/tmp/loom-695-owner.log`.

Three accepted specimens independently generated at exact parent/current/repeat
have 63 artifacts total, byte-identical throughout; no intended valid-output
differences. Each parent/current generated module builds and vets; repeat only
regenerates and tidies because compilation of identical bytes is already proved.
Times: parent 53.32s, current 32.64s, repeat 17.75s. Each temp module requires
verified pushed `v1.10.0-alpha.9.0.20261013020843-063198a2f427` and explicitly
replaces it with the selected source. Pin receipt `/tmp/loom-695-pin.json`;
comparison helper and report `/tmp/loom-695-comparison/{comparison_test.go,report.json}`.

Five CLI specimens reproduce parent admission and candidate rejection before any
`gen` directory exists: equal exclusive endpoints, negative minimum length,
negative maximum length, named numeric contradiction (base minimum10/derived
maximum5), and named length contradiction. Diagnostics match each domain.
Sources, commands and outputs: `/tmp/loom-695-comparison/rejected.py` and
`rejected-report.json`; summary `/tmp/loom-695-rejected-cli.log`. The first harness
attempt required `go mod tidy` before CLI inspection; it was corrected before
recording semantic acceptance. These inputs are durable in the direct/DSL tests.

The prior readiness model sources are unchanged: Local and MergedOnly fail their
intended invariant, Checked passes 1297/865 states. Its bounded arithmetic claim
does not establish arbitrary carrier or pattern satisfiability. No generator
corpus, dependency update, transport presence policy or default application change
is included.
