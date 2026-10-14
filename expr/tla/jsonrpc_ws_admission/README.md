# JSON-RPC WebSocket effective admission

Admission model for #728 and its implemented shared binding/request-source owner.

Before the correction, validation read only authored transport maps at service validation;
endpoint validation resolved implicit bearer/session bindings later. Basic
never enters the header mapping object. Separately, endpoint finalization
replaces Payload with StreamingPayload, potentially discarding credential tags.
The shared service generator omits schemes whose credential tag is then absent.

The model separates authored mapping visibility, resolved mapping visibility,
actual binding location and preservation of its credential source/role. A
server-streaming Payload moved intact into StreamingPayload has source `same`.
A distinct streaming payload missing the credential is `lost`; a member present
without the credential role is `retagged`. `none` includes NoSecurity and anonymous
overrides after the existing effective-scope owner has selected them. Authenticated
OR alternatives are not permission to discard one branch: apply admission to
every binding, including an authenticated branch beside an anonymous branch.

## Run

Pinned TLA+ release 1.7.4, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, for each configuration below:

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config checked.cfg -metadir /tmp/loom-728-model-checked WebSocketAdmission.tla
```

- `legacy`: expect `NoUnsupportedWireBinding` failure for inferred credentials.
- `resolved-mappings`: expect the same failure for Basic absent from header maps.
- `location-only`: expect `NoLostCredential` failure for message credentials lost
  by Payload/StreamingPayload replacement.
- `blanket-security`: expect `PreserveSupported` failure for working message auth.
- `checked`: all invariants must pass.

## Correspondence and limits

`expr/http_wire_bindings.go` owns binding location, ordered sources, scheme roles
and AND/OR membership. `expr/http_endpoint_jsonrpc.go` checks effective locations and every source/tag
role against `jsonrpcRequestPayload`; ordinary mapping policy remains in
`expr/http_service_jsonrpc.go`.
`expr/http_endpoint_finalize.go` performs request-source replacement;
`codegen/service/service_data_methods.go:buildCredentialSchemeData` returns nil
when the final payload has no credential tag. `expr/jsonrpc_ws_security_test.go`
exercises admission through complete DSL. The probe sources here preserve actual
generated-module and mounted-WebSocket evidence, including a silently unprotected
body credential design. Model checks found the expected failures in the raw-mapping,
resolved-mappings, location-only and blanket-security variants after 115, 118,
167 and 124 states, respectively; the checked variant passed all three invariants
in 168 states. Named strings,
renamed/retagged or invalid-type credentials, merged roles and partial AND/OR
loss are covered by the Go admission regressions. Selector identity is checked
before and after repeated normalization/finalization.

The finite model abstracts one selected effective binding, independently of
AND/OR aggregation (production must check all). It assumes source/role identity
has been extracted correctly; Go tests must establish that, including named
payloads, explicit body selection, inherited scopes and mixed transport roots.
It neither proves arbitrary Go types/constraints nor validates browser handshake,
auth callbacks, generators, network or session-cookie security. `retagged` is a
source-role mismatch, not permission to reinterpret a differently authored
credential. Ordinary HTTP/SSE are normal controls outside this WebSocket rule.
Preserve model sources/configs and concise findings; remove checker state/trace
outputs after independent review when no process needs them.
