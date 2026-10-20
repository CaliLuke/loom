# Explicit status-derived error identity

The accepted contract adds an explicit source for a custom required ErrorName.
It does not repair the closed #1027 proposal by substituting a status for a
missing required header. `fallback.cfg` explores that rejected alternative:
even a unique status cannot satisfy an authored required wire mapping.
`explicit.cfg` preserves required wire inputs and admits a status source only
when the effective endpoint has one response at that status, there is no
competing field mapping, and the declared error name satisfies its constraints.

Run with the repository's pinned TLA+ 1.7.4 tools JAR (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config fallback.cfg -metadir /tmp/loom-identity-fallback StatusIdentity.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config explicit.cfg -metadir /tmp/loom-identity-explicit StatusIdentity.tla
```

The fallback must fail `RequiredWireRemainsRequired`; the explicit design must
pass all invariants. This is a finite comparison of policies, not evidence for
product intent or a proof of the generated implementation.

With Java 27, the fallback failed that invariant after 261 states, with a
missing required header at a unique status. The explicit policy passed with
384 states generated and 352 distinct states.

`expr/http_error_name_status.go` owns admission. `http_reconstruction.go` counts
the resolved source, `http_body_types.go` excludes it from implicit bodies, and
the transport IR carries that field to the ordinary error constructor inputs.
`http_error_name_status_test.go` covers rejected, inherited, constrained and
explicit-body designs. `http/codegen/error_name_status_test.go` exercises generated
clients, servers and response contracts, including strict required headers,
renamed Go fields, absent bodies, malformed bodies and unbound extra headers.
`http/codegen/openapi/v3/error_name_status_test.go` checks the documented shape.

The model abstracts field constraints to a Boolean and covers two error
identities plus a success response sharing a status. It does not model Go field
layout, body validation, wire codecs, header spelling, inheritance evaluation,
SSE or WebSocket state. Tests cover implementation details outside that scope.
Remove task-owned checker state and traces after review; retain these inputs.

## Generated-output acceptance

At parent `c8a6d196b`, compare `OpenAPISharedErrorHeaderDSL`,
`DefaultErrorResponseDSL`, `UnionResponseBodyDSL` and `ViewedUnionDSL` with shape
included and untagged encoding. These cover required header identity, built-in
errors, ordinary unions and viewed response declarations. All 195 generated
artifacts, including OpenAPI JSON/YAML, remain byte-identical. Compile both
parent and candidate modules. Add `OpenAPIStatusErrorNameDSL` for the new behavior
and compare two process-isolated generations byte for byte.

Intended changes are limited to opted-in designs: omitted implicit ErrorName
body fields and automatic routing headers, constant constructor inputs, and
response contracts that require no routing header. Existing checked-in HTTP and
JSON-RPC ticktock and mixedtick fixtures change only `gen/loom.json` design
digests because the evaluated expression now includes the new source fields.
Their generated Go and existing wire contracts remain unchanged.
