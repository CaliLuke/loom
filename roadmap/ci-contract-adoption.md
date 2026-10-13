# CI contract adoption — #890

The integration suite must assert the contracts already delivered by the shared
header-list, security-scope and response-body owners. Production semantics and
fixture inputs stay unchanged. CI run 38457192450 identifies the stale consumers.

## Status

- 2026-10-13 — Delivered in `59ac9947c`, independently reviewed, pushed to main and confirmed closed. Production source is unchanged.

### Milestone 1: Restore meaningful integration expectations

Goal: Existing assertions enforce the accepted contracts and retain their original coverage.

Acceptance Criteria

The exact CI failures pass against current local source. Named-array round trips,
scalar/cookie controls, credential values/context/rejection, and mixed-scope identity
remain asserted. Two regenerated OpenAPI goldens add only the three authored empty
body routes and their schema. Independent review leaves no unresolved findings.

Checklist

- [x] Read the framework skill and owning source; recite workflow before edits.
  Root follows reproduction, assertion adoption, focused verification, fresh review,
  atomic commit/push and closure. No production or direct gen edits; preserve local
  work and use explicit LOOM_DIR for generated-module checks.
- [x] Inventory the CI failures and current assertions in client_primitive_fields_test.go,
  named_string_arrays_test.go, credential_fields_test.go, security_credentials_readiness_test.go
  and TestFiles/inline-body-selection; preserve exact CI log at /tmp/loom-ci-888-ci.log.
- [x] Reproduce the named tests on current main before editing their expectations.
- [x] Adopt canonical comma-list framing and per-scheme scopes; promote the former
  mixed-scope counterexample to a positive contract test without deleting its assertion.
- [x] Regenerate only TestFiles/inline-body-selection with -update; inspect JSON/YAML
  changes and preserve existing paths byte-for-byte structurally.
- [x] Run the exact affected HTTP/importer tests, selected OpenAPI test, and touched
  package lint. Existing renderer checks validate both specs with libopenapi.
- [x] Fresh agent review of this exact test-only diff and evidence, then commit,
  push main, close #890 and retain concise evidence before temporary cleanup.

## Evidence

At parent `0df1a49bc`, `/tmp/loom-890-before.log` reproduces every HTTP/importer
failure. `/tmp/loom-890-golden-before.log` separately reproduces both goldens
(the first multi-package regex did not select their slash-delimited subtest).
The updated HTTP cases pass in 22.246s; importer candidate in 10.556s. The exact
OpenAPI subtest passes in 0.370s, including JSON/YAML libopenapi model validation.
Scoped golangci-lint reports zero issues after converting the structural selector
to a switch. Its direct test was rerun after that mechanical edit.

`/tmp/loom-890-golden-delta.json` confirms the only JSON contract additions are
`/empty_error`, `/empty_selected`, `/empty_type` and `EmptyRecord`; all other values
are unchanged. YAML diff has the same additions. Pinned Vacuum v0.33.0 with the
repository's pinned libopenapi override reports zero errors for both artifacts
(`/tmp/loom-890-vacuum.log`; existing description/style warnings remain).
No production or design input changes invalidate previous generator byte/compile
evidence. The existing generated-module tests compile and exercise their unchanged
generated artifacts with corrected assertions; no full corpus audit was run.

Independent final re-review confirmed the corrected former-red-probe comment and
all other exact source/evidence hashes. Receipt:
`/tmp/loom-890-independent-final-rereview-verdict.json`. CI adoption repairs only
the named failures; it does not assert aggregate current-main CI success.
