# API error-response preparation — #912

The current API-owned HTTP response is validated without `HTTPResponseExpr.Prepare`.
Service and endpoint copies are prepared, so a primitive API error panics when
shared response-body analysis reads the original response's missing header map.
Current `ce26ecbe9` reproduces this in the existing APIPrimitiveErrorResponseDSL
encoder test and actual CLI generation. This is independent of #908 runtime error
ownership and #911 built-in error reconstruction.

## Owner and sequence

1. [x] Preserve the existing direct/CLI failures and a diagnostic-only stack probe.
2. [x] Add failing preparation invariants across API/service/method declarations,
   primitive/named/array values and explicit headers, retaining ordinary controls.
3. [x] Add API error-response initialization to `HTTPExpr.Prepare`, which the
   evaluator walks before service inheritance and before validation. Reuse the
   existing response preparation owner; do not add downstream nil guards or
   alter lookup, inheritance, reconstruction or validation policy.
4. [x] Check repeated preparation preserves authored maps and values. Run direct
   expr and existing encoder controls. Generate existing API primitive plus
   ordinary method/header controls from parent/candidate in separate processes,
   compare normal outputs exactly and build/vet candidate modules.
5. [ ] Freeze the exact diff and evidence for independent review, then one commit.

No public semantic change or new formal model is needed: this restores the
existing prepare-before-validation invariant. API and endpoint response records
remain distinct; inherited status/body/header behavior must remain unchanged.
The API-level owner is shared by HTTP and JSON-RPC expression roots; uniform
JSON-RPC response-body admission remains enforced. No generated files are edited.

## Acceptance evidence

Parent `ce26ecbe9` fails four of fifteen declaration/shape cases: primitive,
named primitive and array API errors panic; the API object response retains an
unprepared header map. Eleven service/method/header controls pass. The candidate
passes all fifteen cases, including repeat preparation without replacing authored
maps. Existing response validation, error encoder and JSON-RPC response-admission
controls pass under the race detector; scoped expr lint reports no issues.

Generation used an immutable parent archive and a copy with only `expr/http.go`
changed, excluding concurrent #911 work. Existing `APIPrimitiveErrorResponseDSL`,
`PrimitiveErrorResponseDSL` and `PrimitiveErrorInResponseHeaderDSL` exercise the
actual CLI. The parent API specimen fails during evaluation; both ordinary
specimens build and vet. All three candidate modules build and vet. The 42
ordinary artifacts are byte-identical to the parent, and all 63 candidate
artifacts match a separate generation process. Repeat generation does not repeat
build/vet for identical output. No golden or generated output changes are needed.

The new direct test also passes against the isolated candidate source. This
restoration does not prove every invalid API expression is safe: existing
validation still owns admission, and unrelated error reconstruction stays #911.
