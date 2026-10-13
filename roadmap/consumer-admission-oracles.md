# Consumer contract oracles after constraint admission

Status: [#836](https://github.com/CaliLuke/loom/issues/836) test-only repair and bounded acceptance complete; independent exact-diff review pending. Base: `a8783ef651e7ef433d5724b36e201341c41f8436`.

## Failure and owner

CI on `5e4b4457e` passed #833's repaired byte-schema groups, then exposed three more stale consumer expectations: OpenAPI IR `TestBaselineByteProjectionRenderedConstraints/contradictory interval`, Lean `named-key-contracts/four numeric bounds`, and Lean `effective-alias-lengths/contradictory`. Complete failed-event extraction from the provided value-contract log identifies only the latter two leaf failures. These tests still treat contradictory design bounds as a valid empty runtime domain, contrary to the accepted #695 design-admission contract.

The mathematical reference remains valid over empty intervals. Keep those independent raw-layer/reference checks and distinguish them from source occurrence, effective-constraint and target-plan admission. Do not weaken production rejection or remove test cases.

## Sequence and bounded acceptance

- [x] Inspect complete distinct failure events from `/tmp/loom-ci-80-openapi.log` and `/tmp/loom-ci-80-value.log`.
- [x] Reproduce the exact enabled IR test and both full real-Lean consumer groups at this base; preserve failure logs.
- [x] Prepare a self-contained issue draft; root reviews before filing and releases test-only repair.
- [x] Keep the IR contradictory fixture as an explicit design-rejection assertion; ensure only admitted cases index the ordinary/async schema batches. Preserve all five valid cases and their complete 13-instance matrices in both paths.
- [x] Keep the contradictory numeric key's raw reference contract and endpoint queries; assert production effective-constraint/whole-map admission errors instead of attempting invalid-map value resolution.
- [x] Keep all raw alias length layers and independent reference outcomes. Assert contradictory effective/source/target admission; preserve valid String rune-count, Array and Map resolution/projection cases.
- [x] Update owning correspondence claims. Run focused enabled Ajv, both real Lean groups, affected ordinary package tests, scoped lint and docs. Re-run #833's neighboring focused groups to cover the adjacent split; no full corpus or generated Go changes.
- [x] Freeze exact diff and evidence for independent review; root owns atomic commit and issue closure.

## Model boundary

No production or theorem change is proposed. Existing `expr/tla/constraint_domain` models admission and the Lean arithmetic functions deliberately retain a wider mathematical input domain. Tests must establish correspondence only for admitted designs and assert rejection at other boundaries. This is a repair to executable conformance claims, not a new product policy.

## Parent evidence and independent readiness

The exact IR gate fails in 0.540s. Both complete real-Lean consumer groups fail
in 18.428s at the reported leaves, with no other failures. Logs are preserved
under `/tmp/loom-consumer-admission/{ir-parent,lean-parent}.log`.
Root reviewed the owners, plan and exact issue draft SHA
`fbf078d41bb663d535c19bab8ffde39456f3a5b9ad88e90df9dfdc42630021a7`,
authorized test-only implementation and publication; the approved draft was
filed as #836. The reference uses the unchanged pinned Lean v4.34.1 project.

## Same-owner direct-test follow-up

The first required full IR package run found two additional unchanged direct
oracles with the same admission mismatch. `TestAsyncByteAliasBoundsRemainConjoined`
used only a 3..2 interval; `TestByteAliasBoundsIntersectBeforeProjection` included
MaxInt..2. The isolated tests reproduce both failures in
`/tmp/loom-consumer-admission/additional-parent.log`. Root reviewed and approved
the expanded issue draft SHA `059c7dfb90face0a34d73bcaaa9075e33832209429cd9b5392220c25c480e166`.
Both retain their invalid fixtures as admission rejections. The overflow test
retains its legal narrowing case; the async test now also proves legal inherited
1..2 bounds yield exactly the two four-character base64 residue branches.

## Final acceptance

- Exact enabled IR/Ajv gate: pass 1.862s, 130 admitted ordinary/async instance checks plus explicit contradictory-design rejection.
- Both full repaired real-Lean groups and both #833 byte-reference neighbors: pass 26.075s, with all four executed groups recorded as passed. The wider mathematical empty domains remain exercised.
- #833 neighboring inline Ajv test: pass 2.074s, preserving 29 valid instance controls and invalid-design checks.
- Additional direct async/ordinary rejection and positive projection controls: pass 0.498s.
- Evidence is under `/tmp/loom-consumer-admission/`; parent CI logs and exact issue drafts remain separately preserved. This test-only patch changes no generated artifacts, production code or Lean theorem sources.
- Final full affected packages: IR 6.148s; valuecontract reuses the unchanged valid result from 2.461s. Scoped lint: 0 issues. Docs and whitespace checks pass.
- Exact review inputs: `/tmp/loom-836-final-manifest.json` and `/tmp/loom-836-final.patch`.
