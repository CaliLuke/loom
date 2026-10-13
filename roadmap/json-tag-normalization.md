# Authored JSON tag normalization — #844

Status: implementation complete, awaiting independent final review. No commit yet.
Parent: `03e38fd16` (includes #718 diagnostic wire-name owner).
Issue: https://github.com/CaliLuke/loom/issues/844

## Scope and acceptance

One shared owner must preserve authored full JSON tags and exact exclusion tags
across service declarations, HTTP/JSON-RPC encoders and Extend identity checks.
Preserve current automatic omission, explicit carrier normalization, JSON-RPC ID
policy, custom codec tags and rejected omitted-body designs. Native service JSON
must not disclose an exact name-only `-` field. Generated transport and retained
example projection must agree for the corrected ordinary full-tag cases.

M7 automatic absence-only omission is a separate follow-on. This commit does not
change its policy or introduce custom field wrappers. Full authored `omitzero`
on native scalar values and arbitrary external codec projection remain existing
representation boundaries, not a claim of universal projection equivalence.

## Sequence

1. [x] Reproduce current behavior with direct normalization, actual JSON-v2
   serialization and valid generated service/HTTP/JSON-RPC witnesses.
2. [x] Independently review readiness and exact report; publish approved #844.
3. [x] Advance to #718's parent-compatible main state and reconfirm the reds.
4. [x] Make `internal/encodingmeta.JSONTag` the single normalization owner.
   `JSONTagPolicy` carries caller-owned omission, explicit presence and protocol
   override facts; exact exclusion is handled before adding options. Service and
   Extend consume the same owner. HTTP delegates rather than merging options.
   Delete `representation.MergeJSONOmitOption`, moving its controls to the owner.
5. [x] Check generated behavior, output comparison, compilation, rendered specs,
   deterministic output, affected direct tests and scoped lint.
6. [ ] Obtain fresh exact-diff review; repair findings and re-review.
7. [ ] Root integrates, commits atomically, pushes, closes #844 and cleans
   temporary artifacts after preserving concise evidence.

## Evidence

Go 1.27.2 / GOEXPERIMENT=jsonv2; generated-module helper pins pushed
`063198a2f427` and local verification explicitly sets LOOM_DIR. The source
boundary changed only in generator/expression metadata code; no runtime package
changed. Parent generation uses an overlay with the four exact production
owner files from `03e38fd16`; test harnesses and new specimen inputs are identical.
Generated parent modules compile against the same unchanged runtime sources.

- Parent witnesses: two ordinary full tags are rewritten; full-tag HTTP/RPC
  requests and responses lose all five supplied empties; generated native skip
  emits `{"-":"private"}`. Earlier exact report snippet also fails all three
  assertions. Controls preserve populated/absent values and custom time.Time.
- Final direct/transport checks: codegen 0.195s, expr 0.201s, HTTP codegen 4.085s,
  generator 15.706s, encodingmeta 0.263s. Covers full/no-option/omitzero/omitempty,
  exact skip, valid literal dash, precedence, explicit empty tag, nullable default,
  JSON-RPC ID overrides, actual nil/empty wire bytes, composition both orders,
  existing skip admission, generated wire and #718 diagnostic controls.
- Four bounded generated specimens: ordinary/custom unchanged control, full-tag
  HTTP/RPC, native skip, and the existing scalar-map OpenAPI specimen extended
  with authored tags. 80 generated artifacts per parent/candidate/repeat, 71
  unchanged and nine intended changes. Candidate/repeat are byte-identical.
  Every distinct parent and candidate module builds and vets (eight modules).
- The nine differences are six HTTP/RPC encoder type files retaining full tags,
  two CLI example files retaining selected empty members, and one service type
  changing name-only exclusion from `-,omitempty` to exact `-`. There are no
  constructor, decoder, validation or custom codec changes.
- OpenAPI JSON/YAML response examples change from `{}` to
  `{"items":[],"plain":""}`. Declared schema and decoder request examples
  retain all authored members, including explicitly omitempty `omitted`, because
  those positions describe admission rather than response encoding. Both formats
  parse/build with libopenapi and pass pinned Vacuum (9.123s). Separate process
  candidate/repeat bytes match exactly; parent assertions reproduce the loss.
- No checked-in HTTP/JSON-RPC fixture design authors the affected full/name-only
  metadata (bounded search excluding generated trees). The unchanged ordinary
  generated specimen verifies default tags; no checked-in gen files were edited.
- Scoped golangci-lint reports zero issues for owners/consumers and testdata.
  `git diff --check` passes. This is a bounded ticket gate, not a full corpus audit.

Receipt paths: `/tmp/loom-844-artifact-comparison.json`,
`/tmp/loom-844-generated.diff`, `/tmp/loom-844-spec-comparison.json`,
`/tmp/loom-844-final-tests.log`, `/tmp/loom-844-parent-build.log`,
`/tmp/loom-844-candidate-build.log`, `/tmp/loom-844-spec-candidate.log`,
`/tmp/loom-844-lint.log`, `/tmp/loom-844-testdata-lint.log`.
Final source/evidence hashes are frozen in `/tmp/loom-844-final-manifest.json`.

The literal dash wire control uses valid `-,omitempty`. JSON v2 rejects trailing
comma `-,` as malformed; direct option-normalization controls preserve its bytes
without asserting codec admission. The normalizer does not broaden JSON syntax.
The existing M7 model/readiness witnesses remain worktree-local and are excluded
from this atomic diff. Its generated witness now reuses this commit's identical
ordinary-control design/harness; the original source was preserved at
`/tmp/loom-json-omission-generated-witness-before-reuse.go`.
