# Imported parameter admission (#791)

## Owner and contract

The importer must publish only bindings supported at their wire location. The
shared `planDocument` admission pass runs for Analyze and direct Render, before
render planning and name publication. Correct its schema-to-wire shape adapter
and replace the cookie-only admission decision with one location-aware rule.
Keep rendering policy-free.

The existing `urlShape` labels every array a scalar sequence and every memberless
object with `additionalProperties` a map, including closed records. Consume the
#776 object classifier, resolve references with the existing canonical JSON
Pointer lookup, and inspect the resolved array element. Query serialization still
belongs to `httpencoding.Match`: scalar text, repeated scalar arrays, and named
JSON maps including nested values. Fixed query records remain rejected. Header
and path bindings accept primitives and primitive arrays; cookies accept
primitives. Existing Any handling in non-query primitive codecs remains intact;
this repair does not reinterpret unconstrained query/form serialization.

Header/cookie map acceptance in expr is not evidence of a supported codec:
`header_conversion` has scalar/Any/array branches but marks maps unsupported.
Preserve scalar/list header behavior from #689. Response headers consume the same
location rule as request headers. Raw request bodies and form roots retain their
existing admission boundaries; form fields consume the corrected shape adapter.

## Sequence

- [x] Recover existing five failing admissions and eleven passing controls from
  `/tmp/loom-791-current-evidence.json`; preserve its actual generated header
  failure rather than repeat that module build.
- [x] Deliver #776 first, then cleanly reuse its managed worktree at `3476cc239`.
- [x] Add bounded direct regressions for existing failures, closed query records,
  equivalent free maps, referenced elements/parameters, response headers,
  selection, direct Render and blocking allow-lossy behavior.
- [x] Review this owning proposal and parent failure evidence before production.
- [x] Implement one planner binding decision and shared shape/reference adapter;
  remove the cookie-only competing decision. No transport renderer patch.
- [x] Run direct tests and existing query/form controls. Compare parent/candidate
  ordinary scalar/list and JSON-map outputs using existing importer generation
  helpers; compile unique outputs, repeat generation byte-for-byte, and validate
  their OpenAPI specs. Record every intended difference.
- [ ] Freeze exact final diff/evidence for independent review and root delivery.

## Readiness evidence

On parent `3476cc239`, the new 31 direct cases have 14 expected failures and 17
passing controls (`/tmp/loom-791-parent-admission.log`, 0.646s). They include the
original five failures, closed-versus-open query mapping, referenced array items,
request and response headers, and a manually modified Document bypass attempt.
Each rejection checks Analyze under allow-lossy and direct Render, with an
occurrence-qualified diagnostic when available. Selection of an unrelated valid
operation still succeeds.

Existing query/form spelling and raw-form controls pass in
`/tmp/loom-791-parent-controls.log` (18.419s, including the two existing raw-form
module checks). The historical generated header-object failure remains in
`/tmp/loom-791-current-evidence.json`; it needs no duplicate generator run during
readiness.

The implementation will use existing primitive admission for non-query fields
(including existing Any codecs), scalar-element classification for sequences,
and #776 for object carriers. Query/form serialization keeps its existing
unconstrained-schema boundary. Invalid shapes get one owning binding diagnostic
rather than a second serialization error for the same unsupported carrier.
Reference lookup must share component identity and retain cycle protection.
The original slash/tilde component probes were invalid OpenAPI inputs; corrected
positive controls use legal component names. Percent-encoded legal references
are a separate #891 obligation.
No change to the existing union primitive-admission owner is required.

## Bounded evidence and limits

This is a deterministic location-by-shape admission table, not a concurrent or
stateful protocol. Existing #776 constructor/carrier model remains unchanged;
direct table tests cover this additional admission predicate and actual renderer
integration. No new formal state machine or full corpus audit is warranted.

Reuse `TestUniformJSONReadinessImport`, `TestUniformJSONReadinessFormImport`,
`TestUniformJSONReadinessImportedMapCompiles`, raw-form tests and the existing
parameter-reference/selection helpers. Keep numeric domains, annotation loss,
byte-length semantics, typed-map inheritance and unrelated expr admission defects
separate. No public policy decision is outstanding.

## Implemented candidate and final evidence

The shared `admitParameterShape` admits request parameters and response headers
before render planning. Unsupported carriers receive one `parameter-shape`
diagnostic; query serialization runs only for an admitted carrier. `urlShape`
uses the existing component-reference owner, checks primitive array elements,
and consumes #776 object classification. No exporter, HTTP runtime, transport
generator, or union primitive policy changed.

All 34 direct cases pass with race detection (2.097s), including three explicit
Any non-query controls and the corrected legal-name reference controls. Earlier
unchanged query/form, cookie/reference, scoped lint and docs checks remain valid.
Three corrected generated binding specimens and the existing mounted JSON map
roundtrip pass together (31.403s).

Actual parent `3476cc239` and candidate production were generated from separate
source trees with test-only capture overlays. All four candidate modules produce
64 artifacts, exact across independent regeneration. Ordinary scalar/list, named
map alias and existing JSON-map specimens have byte-identical parent/candidate
DSL and artifacts. Only the omitted-open JSON map is newly admitted: its 16
outputs are listed in `/tmp/loom-791-legal-artifact-comparison.json`. The parent
rejects that input before rendering; the other three parent modules compile.
All four unique candidate modules compile and pass vet. All eight generated
JSON/YAML specs pass libopenapi parsing/model construction and pinned Vacuum
contract checks. No spec failure is waived.

## Invalid specimen correction and limits

Independent #891 assessment established that `Item/Name`, `Item~Name`,
`Item/Value` and `Filter/Map` violate the OpenAPI Components key pattern. The
prior spec failures and exact specimens remain archived under
`/tmp/loom-791-invalid-specimen-packet`; they demonstrate permissive invalid-input
admission, not supported escaped-component roundtrips. Their generated artifacts
and logs are historical evidence, not positive acceptance evidence. The earlier
classification of these failures as a supported exporter contract was wrong.

The positive matrix now uses legal `ItemName`, `ItemValue` and `FilterMap` with
ordinary references. It retains referenced scalar/list/map, omitted-open map,
selection and direct-Render obligations. Invalid component names should be
rejected at component admission; that separate defect is not repaired here.
Legal percent-encoded references remain under #891's shared identity correction,
and are not claimed supported by this packet. No change to JSON Pointer property
names or component-name policy is included.

The exact final source, commands, current artifact ledger and historical evidence
disposition are frozen in `/tmp/loom-791-final-manifest.json`. Independent final
review and root integration remain pending; no full-corpus audit is claimed.
