# Diagnostic redaction projection and lifetime

This finite design model covers the shared policy required by #718 and #721. `NamesOnly` models
historical parent diagnostics, which have no authored credential metadata and miss camelCase
names. `LogicalName` tests an insufficient repair that masks physical query names
but reuses logical source names for remapped JSON fields and cannot represent
scalar-body or path credentials. `Borrowed` additionally models application-owned
mutable policy storage retained until a response is consumed. `Snapshot` derives
all physical locations and owns immutable state before observation.

The output property is deliberately narrow: the modeled credential cannot appear
in a completed diagnostic. Wire preservation, ordinary-field visibility, parser
behavior, complete/incomplete previews, header casing, and actual generator wiring
are implementation test obligations. The model does not establish that arbitrary
unmodeled application data contains no secrets. It abstracts no-byte or masked
output equally as `masked`; product policy determines which representation is used.

From this directory, run each configuration using the pinned shared TLC jar and an
external metadir:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-718-model-Snapshot -config Snapshot.cfg DiagnosticRedaction.tla
```

`NamesOnly`, `LogicalName`, and `Borrowed` must fail `NoCredentialDisclosure`;
`Snapshot` must pass. `MutateCaller` occurs after capture but before completion to
represent the delayed EOF/Close logging introduced by the passive response owner.
An immutable service aggregate also avoids request-context publication ordering.
Implementation correspondence: `diagnostics.New` copies caller metadata;
`Policy` owns conventional and explicit masking. Client exchange capture retains
that immutable policy through EOF/Close, while server Debug uses the same owner.
`expr.HTTPEndpointExpr.CredentialBindings` projects identities from the effective
binding plan, including explicit duplicate physical operations. The generator's
prepared body projection uses emitted JSON/form tags, not logical member names.

`http/diagnostic_redaction_contract_test.go` covers conventional names, immutable
metadata and delayed completion. `http/middleware/debug_redaction_test.go` covers
wire preservation, nested bodies and route-variable masking. Generator tests in
`http/codegen/diagnostic_redaction*_test.go` and
`codegen/generator/diagnostic_credentials_test.go` connect source roles to generated
HTTP/JSON-RPC metadata and example wiring. The finite model does not enumerate
DSL legality, duplicate Basic transport slots, codec failures or HTTP I/O; those
are direct/runtime/generated obligations.

Access-log correspondence (#721): `clue/log` constructs immutable policies in
`WithURLCredentials` and `WithClientURLCredentials`, before a middleware/client
is constructed or an option is reused. Each option contributes its policy;
sequential URL projection cannot restore a previously masked location. Default
access logging uses the conventional projection. `clue/log/access_credentials_test.go`
covers both server log events, client completion/failure, caller mutation, option
reuse/composition, malformed queries and unchanged requests. Generated mixed and
disjoint hosts in `codegen/generator/access_log_credentials_test.go` execute the
emitted middleware configuration and generated credential metadata. Host selection
and ordinary query visibility remain implementation obligations rather than new
properties of this finite model.
