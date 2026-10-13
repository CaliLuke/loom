# Map-key admission — #702

Implementation on delivered `22dddd67f` after independent readiness approval. The reported native Bytes and array keys pass evaluation, then panic
or produce invalid Go. The owner must reject unsupported framework-owned key
representations before value capture and rendering without rejecting documented
object keys or opaque user codecs merely because their semantic kind is broad.

1. [x] Reproduce native and named Bytes/array keys at evaluation and actual
   generation; retain ordinary, object-pointer, Any and custom-key controls.
2. [x] Trace prepared type references, occurrence metadata, shared validation,
   physical key rendering and generated-example ownership. Classify independent
   residual failures instead of masking them with blanket primitive-only rules.
3. [x] Specify one shared admission owner, supported/rejected examples and
   boundary to authored custom Go representations. Model ordering/alias reuse
   only where it adds evidence beyond direct classification tests.
4. [x] Freeze additive regressions, source evidence and a bounded acceptance
   matrix for independent readiness review.
5. [x] Implement only the approved owner after readiness approval.
6. [ ] Compare representative parent/candidate/repeat outputs, compile and test
   affected consumers, then obtain exact final review before one atomic commit.

The public `MapOf` example uses an object `Bottle` key. Go service rendering uses
object pointers; that contract must not be erased. `Any` keys deliberately use
`any` instead of `loom.JSONValue`, covered by existing compiler tests. Custom
`struct:field:type` names have an external representation contract; inspection
must distinguish that boundary from native slice storage. gRPC's narrower map
key domain and JSON object-name codec admission remain separate owners.

## Selected owner and scope

The authoritative boundary is the map occurrence in the root Go-representation
traversal. `eval.RunDSL` prepares every root and its sets, calls `RootExpr.Validate`
(which owns this traversal), then validates child sets and only finalizes when
all validation succeeds. Thus admission precedes finalized occurrence capture
and generated examples. The traversal includes unused types and errors; these
bypass complete attribute validation because `AttributeExpr.Validate(ctx,parent)`
is not an `eval.Validator` and error validation checks only its own contracts.
The map check lives only in this traversal, avoiding duplicate diagnostics from
root and child-set validation. Inspect its selected `KeyType` representation,
not the key type's ordinary validation method: ordinary Bytes/arrays stay valid values and the same
named declaration may appear in both roles. Reject framework-owned Bytes, array
and map storage, following named declaration edges. Preserve object pointer
keys even if their objects contain slices, `Any`'s deliberate `any` key carrier,
and scalar/native named keys.

Use the existing positional `encodingmeta.GoType` reader when a primitive
representation actually accepts a `struct:field:type` override. Leave that
explicit representation's comparability to its authored Go contract. A metadata
override on a collection or a named occurrence does not replace the generated
collection/named declaration; those cannot bypass native slice admission.
Do not parse arbitrary Go syntax, import custom types, invoke codecs or introduce
a general comparability API. Existing nullable-key validation and transport
restrictions remain. Delete the `MapOf` map-kind policy check and its misleading
message that explicitly endorses arrays; argument/type lookup errors remain DSL
construction errors and map-key admission has one expression owner.

This fixes a class across method payload/result/stream/error, unused named types,
nested map values and arrays of maps. It does not add per-transport checks or
change generated representations. Routine valid generated output must remain
identical. No public custom-Go-type policy is proposed.

## Reproduced boundary and residuals

On `22dddd67f`, 18 direct native/named/context/root cases still fail their expected
admission assertions; 11 valid admission controls and 6 physical type controls
pass. Four actual CLI cases reproduce two late Bytes panics and two successful
array generations that should have been rejected. Their current generated key
carriers are slices. The native Bytes panic occurs in `Map.Example`, before any
useful generated API is published. The test requires a source-qualified
`eval.RunDSL` map-key diagnostic and no `gen/` output after the repair.

One generated service module proves that String, Boolean, Float64, named String,
object-pointer keys (including a slice field), Any and a comparable `[16]byte`
authored override retain valid Go declarations and complete build/vet. Its
custom byte-key map supplies an authored empty example. Without that example,
custom Bytes key synthesis still panics even though the physical key is
comparable; that separate generated-example residual is retained in
`/tmp/loom-702-inspect.log`, not claimed fixed or used to reject the custom type.
An authored String override to `[]byte` likewise remains the author's external
Go-type responsibility; this ticket does not inspect opaque representations.

The small model checks one shared declaration in both key/value visitation
orders. Current admission and a repair behind type-global memoization fail
`KeySafety`; primitive-only admission fails `PreserveContracts`; the selected
map-occurrence owner passes both. This directly informs where the check belongs,
without modeling unrelated transports or all of Go.

## Bounded acceptance after approval

1. Make the 18 evaluator and four actual CLI rejection assertions pass, preserving
   the 11 existing valid admission and six physical-representation controls.
2. Retain existing map/nullable rejection tests and run affected expr/DSL tests.
   Verify a direct constructed map-key map also reaches the shared owner after
   the construction-time policy is removed.
3. Regenerate the single saved seven-shape service control at parent, candidate
   and one independent repeat; compare every generated byte and build/vet parent
   and candidate. Reuse the existing `TestGeneratedAnyMapKeyBuilds` as the bounded
   HTTP consumer check; no new transport harness or full design corpus.
4. Run scoped lint, update the MapOf documentation and maintainer owner guidance,
   freeze final diff for independent review. The tiny model's unchanged results
   can be reused unless implementation changes its context assumptions.

The readiness packet contained only additive tests, this plan and the model.
Root independently approved it before production changes; verdict:
`/tmp/loom-702-root-readiness-verdict.json`. Implementation now uses
`validateGoRepresentation` before finalization/capture and removes the old
DSL map-kind policy. Each rejected specimen asserts one diagnostic, preserving declaration/path
context. Method and named-type diagnostics retain their existing file/line
provenance. `ErrorExpr` has no retained DSL source in the current evaluator, so
its diagnostic retains the error name without claiming a source location. No generated representation or external codec changed.
Final bounded evidence: full expr/DSL tests pass (1.061s/0.248s); four actual
CLI rejections plus the generated seven-shape control pass (14.858s), physical
carrier tests pass (0.332s), and existing Any-map HTTP compilation passes (6.659s).
All four parent/candidate/independent-repeat artifacts match byte for byte using
identical invocation headers. Parent and candidate modules build and vet; no
rendered differences are intended. The checked-in ticktock/mixedtick designs
contain no maps, so this admission-only repair does not regenerate them. Scoped
lint reports zero issues. Models are unchanged and their approved evidence is
reused: traversal still checks each map before visiting a possibly cached key.
The constructed map-key-map witness now exercises root admission directly;
restoring the parent root/traversal in a direct-test overlay reproduces its red.
That overlay is not used for CLI or generated-output evidence.

The exact final packet is frozen for independent review. No commit or delivery
is claimed until that review and root integration finish.
