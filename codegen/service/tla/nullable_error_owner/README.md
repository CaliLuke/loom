# Nullable error ownership

A declared failure has a nominal owner independently of absent, null or concrete
payload data. The former payload-as-owner representation could not implement Go
error for Nullable values. Concrete ErrorName agreement is checked without
rewriting the payload; JSON-RPC accepts a decoded error only for one candidate.

Run explicitly with TLA+ tools 1.7.4 (TLC 2.19), jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -metadir /tmp/loom-nullable-error-model -config nominal.cfg NullableErrorOwner.tla
```

Run each configuration with a distinct temporary metadir. No model runs in CI.

| Configuration | Expected exit / property | Generated / distinct states |
| --- | --- | --- |
| nominal | 0, all listed invariants | 384 / 384 |
| payload-owner | 12, NamedFailure | 109 / 109 |
| discard-selector | 12, ConcreteIdentity | 295 / 295 |
| collapse-null | 12, PresencePreserved | 253 / 253 |
| first-match | 12, ExactlyOne | 330 / 330 |
| bodyless-all | 12, MetadataConcrete | 219 / 219 |

The two names and three presence states are bounded abstractions. Go type names,
pointer layouts, custom codecs, validation and physical envelope bytes are not
modeled. A bodyless declaration without metadata can retain absent data; absent is never changed
into explicit null. `selector = "none"` means no authored selector contract, not
an empty authored field. Matching/missing concrete selectors are tested in Go.

`codegen/service/nullable_error.go` and `BuildErrorDescriptor` own native identity
and checked payload construction/projection. HTTP response constructors and
encoders consume those facts. `jsonrpc/error_decode.go` retains exact-one matching
with a captured nullable candidate fact, while `jsonrpc/types.go` preserves data
presence. `TestNullableErrorDescriptorOwnsFailure`,
`TestNullableErrorIdentityAndOrdinaryResult`, `TestNullableErrorGenerated`,
`TestJSONRPCNullableErrorGenerated`, `TestDecodeNullableErrorIdentity` and
`TestErrorEnvelopeDataPresence` cover direct, compiled and wire correspondence.
The separate response projection model owns nullable parents split across body
and transport metadata. Models do not prove generated syntax or runtime safety.

A bodyless mapping with headers or cookies still projects members and therefore
requires a concrete parent. `metadata` models that distinction; `bodyless-all`
reproduces treating every bodyless mapping as permission for absence.
`MetadataConcrete` rejects that policy. The response projection model owns the
corresponding client reconstruction and supplied/default metadata distinction.

## Mutable native default occurrence

`NativeOccurrence.tla` compares the selected pointer-backed owner with the
rejected proposal that returns an address of Nullable.Value's copied struct.
This is not a claim that the former uncompilable default-error generator executed
that proposal. Application mutation is sequential; concurrent writes and Go
syntax are outside the model. Presence and declaration arbitration remain with
NullableErrorOwner above.

Run the same pinned jar with `-config native-owned.cfg NativeOccurrence.tla`.
`native-copied.cfg` must fail MutationVisible. The reach-message and reach-identity
configs deliberately fail their reachability predicates, showing both mutation
paths execute. The candidate checks observed mutation, identity revalidation and
no invented native occurrence. `renderNullableDefaultError`,
`ResolveErrorResponse`, `TestResolveErrorResponseNativeOccurrence`,
`TestNullableDefaultErrorGenerated` and `TestNullableDefaultJSONRPCGenerated`
check real pointer/cause/remedy ownership and HTTP/RPC publication. A generated
constructor never accepts a pointer payload, so it cannot create a present nil
pointer; defensive private-storage guards also reject that state.
