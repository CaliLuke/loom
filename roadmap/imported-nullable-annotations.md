# Imported nullable annotation ownership (#677)

Parent: `1d03a9e9b`. Owner: normalized schema annotation composition in
`internal/openapiimport`. Nullable anyOf selects the non-null structural schema
by copying it over the outer schema; currently only description is restored.
Move supported annotation application after structural normalization, so type
selection cannot erase title/deprecation/read/write intent. Keep one owner for
ordinary and nullable schemas, not per-renderer restoration.

Sequence:

1. Reproduce outer title/deprecation/readOnly/writeOnly loss with actual Analyze
   and Render over supported fixtures. Cover non-null branch annotations, nested
   nullable composition, reference and unconstrained branches, and rejected
   annotated null branches. Preserve direct scalar and other composition controls.
2. Independently approve the bounded readiness packet before production edits.
3. Apply outer title/description precedence after composition and retain positive
   deprecated/readOnly/writeOnly annotations from either applicable layer. An
   outer false annotation does not erase a positive branch annotation. Merge
   vendor-extension keys deterministically with outer-key precedence, copying
   maps rather than changing shared branch state. Do not change validation/default
   composition, null-branch admission or supported dialects.
4. Verify normalized data, rendered DSL, generated compilation and OpenAPI
   request/response projection. Compare a bounded existing normal fixture at
   parent/candidate, plus annotated nullable variants; record intended differences
   and independent-process repeated generation. Validate rendered OpenAPI with
   existing parser/lint tooling. No full exported-design audit.
5. Document annotation precedence/migration; exact final independent review,
   atomic commit/push and closure. Root coordinates shared skill edits.

This is a deterministic normalization correction; direct structural, render and
consumer tests establish the contract. No new concurrency or protocol lifecycle
is introduced. Existing annotation and value-domain owners remain controls.

## Acceptance evidence

- Independent readiness: `dependency_strategy_review` inspected the shared
  overwrite cause and approved the annotation owner before production edits.
- Nine-case Analyze/Render matrix covers outer flags/labels, inherited flags,
  conflicting labels, nested nullable, references, unconstrained values and
  extension-key precedence. Parent loses outer annotations and branch extensions;
  candidate passes with a detached-map mutation control.
- Exact parent `1d03a9e9b` archived in `/tmp/loom-677-parent-source`; ordinary
  and nullable variants reuse `supported.json`. Both generated modules compile.
  Ordinary16artifacts are byte-identical; nullable16artifacts change only
  `http/openapi.json`, `http/openapi.yaml` and `loom.json`. The intended schema
  delta is readOnly on id, writeOnly on secret, deprecated on nickname, plus their
  three outer titles; the design digest changes accordingly. No Go output changes.
- Independent CLI processes with identical `loom` command arguments reproduce
  all32candidate artifacts exactly. `/tmp/loom-677-artifact-comparison.json`.
  An initial mismatched CLI basename changed generated command comments; that
  comparison was rejected and repeated with identical invocation inputs.
- Generated specs parse/build with the pinned libopenapi. Pinned Vacuum0.33.0
  with the repository's existing enum-fix tooling override passes the nullable
  specimen: zero errors,19warnings. The ordinary specimen is unchanged.
- Focused annotation, strict null-branch/dialect admission and ordinary annotation
  round-trip races pass (8.606s). Existing canonical-preservation and noncanonical
  request/response field-pruning controls are checked separately: imported named
  components intentionally retain read/write annotations rather than pruning
  their canonical schemas. Package lint reports zero issues.
- Logs: `/tmp/loom-677-{parent,base-generation,candidate-generation,final-focused,
  projection-controls,vacuum,lint}.log`. No full exported-design audit was run.
