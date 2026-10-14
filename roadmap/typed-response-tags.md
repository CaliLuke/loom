# Typed response Tag binding — #760

Implementation starts on `5876ada4e`, consuming the root/projection readiness
packet `/tmp/loom-760-readiness/manifest.json`. The documented contract identifies
an existing String-kind result field, including named String types. The first
matching tagged response wins; the unique untagged response is the fallback.

The parent compiler probes expose one missing binding: Tag is carried as two
strings, then response analysis guesses a Go field and pointer state and client
renderers assign an untyped temporary. Missing/non-string fields, renamed/mapped
fields, named aliases and nullable/viewed carriers fail at different consumers.

1. [x] Resolve and validate the semantic result field at response admission.
   Retain its logical key and declared attribute with the authored tag value;
   missing and non-String-kind fields fail with response-qualified diagnostics.
   Named String fields remain supported. Shared IR carries this binding instead
   of requiring consumers to rediscover field identity.
2. [x] Lower the bound field against the actual service or projected result
   declaration once in HTTP response analysis. Use its canonical field name,
   actual scope, field layout and presence representation. Produce the typed
   server predicate and client reconstruction together. HTTP and JSON-RPC
   renderers consume that plan; remove their separate pointer/literal guesses.
   Absent or null values never match; selected responses reconstruct a supplied
   typed value. Do not change field constraints, views or codec policies.
3. [x] Add direct admission/binding regressions before production changes. Reuse
   the existing response-selection and result Tag fixtures for generated builds
   and mounted status/value checks. Cover native required/optional, aliases,
   nullable, renamed, mapped, viewed and bodyless carriers, plus early rejection.
4. [x] Preserve response order and fallback. The existing
   `http/codegen/internal/transportir/tla/response_order` model abstracts matching
   into a set and excludes Go predicate evaluation; this repair changes only that
   typed lowering. Reuse its order tests without changing its selection state or
   adding a redundant model. New direct/mounted tests establish lowering behavior.
5. [x] Compare bounded affected parent/candidate outputs and independent-process
   repeats; compile changed outputs and retain ordinary controls. Record every
   intended change and validate affected specs. Keep #780 mapping coverage,
   #774 publication and unrelated response metadata defects separate.
6. [ ] Document the typed-field contract and regeneration, freeze the exact final
   diff for an independent nonimplementer review, then let the orchestrator deliver
   one atomic commit. Preserve concise evidence before inactive-artifact cleanup.

No public policy question remains: this applies the documented String-field
contract to its legal physical carriers. No per-template validation, importer
flattening or rejection of legal nullable/named fields is an acceptable fix.

## Composition and protocol scope

`HTTPResponseExpr.Dup` is used by inherited error responses. `Response` permits
success declarations only at method scope; API/service success probes reject the
missing error name before inheritance. The copy helper therefore does not drop a
Tag from a supported inherited success declaration. Its unrelated error metadata
copy policy is unchanged. Evidence: `/tmp/loom-760-dup-scope_test.go` and log.

JSON-RPC retains its existing first-response decoder contract, not HTTP status
selection. Its renderer consumes the same typed assignment plan; direct decoder
checks cover native, pointer and projected assignments. The ordinary generated
JSON-RPC module remains a compilation/runtime control.

## Final acceptance evidence

The shared semantic owner now rejects all six HTTP/JSON-RPC missing, integer and
boolean counterexamples. `TestResponseTagBindings` checks declared attribute
identity and the actual named/projected/renamed/mapped/nullable lowering. The
single generated carrier specimen covers 11 methods and 35 mounted states:
matching/nonmatching/supplied-absence plus explicit null on nullable carriers.
Bodies deliberately exclude the selector, so client reconstruction is checked
independently of body decoding. The existing overlapping-tag mounted test retains
first-match and fallback evidence.

Full expr/dsl/transport IR race checks passed (7.849/1.387/1.323 seconds).
Focused HTTP/JSON-RPC race checks passed (8.232/3.981 seconds); the latter includes
real ordinary generated JSON-RPC array/object compilation and runtime behavior.
The existing HTTP tag-string and required-tag goldens remain byte-identical.
Scoped lint includes the new testdata fixture and reports zero issues after a
mechanical fixture-slice preallocation correction. No generator logic changed
for that correction; a final independent capture retained identical bytes.

The exact `5876ada4e` Git archive reproduces compiler failures for the combined
legal carrier specimen. The same archive builds and vets both ordinary Tag
specimens. Its test module resolver needs Git metadata only to locate the
archive; `GIT_WORK_TREE` points at that archive and `LOOM_DIR` pins its source.
An initial missing-Git capture attempt is excluded from passing evidence.

Parent/candidate comparison covers 43 generated/spec artifacts: 28 ordinary
artifacts are unchanged. Only the combined specimen's client and server
`encode_decode.go` files differ. Intended changes are actual renamed field
selectors, mapped-field pointer storage, nullable Value/SetValue operations,
and nominal alias temporary types (including the projected alias). All other
13 carrier artifacts, including OpenAPI, are unchanged. All 43 candidate
artifacts match process-isolated repeats exactly. The three rendered specs pass
libopenapi parsing and pinned Vacuum contract lint. Candidate carrier compilation,
vet and race runtime checks use the same design/module/source as the capture;
identical repeated outputs are not rebuilt redundantly.

Evidence is frozen in `/tmp/loom-760-final-manifest.json`, with exact source,
owner, input and artifact hashes. Failed fixture setup (Type declared inside
Service), an early projected-type assertion and an empty filtered test run are
explicitly excluded from final acceptance; corrected checks are retained.
No full fixture corpus or aggregate CI-green claim is made. Delivery remains
pending independent review and the orchestrator's atomic integration commit.
