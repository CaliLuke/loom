# Access-log credential projection — #721

Parent readiness: `03e38fd16e0eace6b0911f28981b808ab06e9608`.
Implementation parent: `0f1fc3dc3` (fast-forward preserving the readiness inputs).

1. Reproduce server start/end and client success/error URL disclosure with
   unchanged request bytes. Inspect the documented credential boundary, generated
   example wiring and newly delivered shared diagnostic policy.
2. Propose one URL presentation owner: `observability/diagnostics.Policy.URL`,
   already used by development previews, also consumed by clue access logging.
   Preserve ordinary query values; mask conventional credentials by default and
   authored names through the same generated service Fields metadata. Do not
   invent another credential-name list or independently inspect DSL security.
3. Independently review readiness before implementation. Public docs warn against logging Authorization/Cookie metadata, and #721
   explicitly requires protecting authored query credentials. This proposal keeps ordinary
   query visibility and fulfills that existing contract; it does not introduce
   an all-query suppression or allow-list policy.
4. Add constructor options for immutable authored credential metadata to request
   and client logging. Existing generated server examples must pass the union of
   credential metadata for the HTTP and JSON-RPC services they actually host.
   Use one shared example-server metadata/wiring owner, including the existing
   JSON-RPC server extension. Generated CLI Debug wiring already passes Fields;
   do not add an unrelated client logging layer.
5. Exercise conventional/custom/repeated/escaped query values, malformed query,
   userinfo and authored credential paths, ordinary URL controls, all log events,
   option reuse/mutation and original request preservation. Compare bounded mixed,
   single-transport, disjoint-host and no-credential examples at parent/current/
   repeat; compile/vet all changed examples. Transport/schema artifacts should
   remain byte-identical. Reuse the shared diagnostic model's snapshot/projection
   assumptions and extend correspondence tests at this new consumer boundary.
6. Update public logging usage and shared contributor guidance. Obtain exact
   independent final review, integrate, commit/push atomically and close #721.

Scope is URL credential presentation in clue access logging and generated example
wiring. Response-body logging, access-log timing, filters and status policy remain
separate owners. Readiness was independently approved before production changes.

The existing HTTP guide prohibits recording credentials; #721 includes the
conventional `key` leak and custom DSL query aliases. Existing log tests expect
ordinary `q=1` to remain visible. Shared Fields and Policy already define immutable
union, case/name normalization, malformed-query omission, userinfo masking and
whole-path omission. Consume that existing contract rather than creating a
second default visibility policy.

## Readiness evidence

- Parent direct regression: all 12 combinations (four URL forms across server,
  client success and client failure) fail for disclosed credentials. Ordinary
  `view=full`, original URL and invocation checks pass.
- Parent generated mixed HTTP/JSON-RPC example builds and vets before failing
  the absent credential-policy wiring assertion. Existing `DiagnosticCredentialsDSL`
  supplies non-conventional authored credential names.
- Reuse `observability/diagnostics/tla/redaction` rather than adding another
  security policy model: its NamesOnly and Borrowed counterexamples describe the
  missing source projection and mutable-policy alternatives. Snapshot is the
  intended design. Final runtime tests must exercise this new consumer and its
  immutable option lifetime; model correspondence documentation will be updated.
- Extend the shared HTTP example-server owner, which is already the JSON-RPC
  example base. Select credential metadata from both transport expressions for
  the services in the current `Server` only, allocate imports through NameScope,
  and emit a single configured `log.HTTP` call. No service security re-inference.
- Constructor options accept existing diagnostics Fields and freeze through
  diagnostics.New when the option is constructed, permitting safe option reuse.
  Multiple options must compose their field sets rather than overwrite earlier
  masks; implementation must retain immutable ownership throughout composition.
- The final generated witness must execute requests through the generated
  metadata with conventional and custom credentials, not merely assert a
  substring. Include no-auth and disjoint-host controls; ordinary wire bytes and
  transport/schema output remain unchanged.

Readiness logs: `/tmp/loom-721-parent-red.log` and
`/tmp/loom-721-parent-generated-red.log`. These failing probes precede production
edits; final acceptance will expand them to the complete contract above.

## Implementation and acceptance

- `clue/log` projects request URLs through `diagnostics.Policy.URL` before storing
  the URL log field. Server start/end share that snapshot; client success,
  transport-failure and error-status entries use the same owner. No transport
  request is changed and ordinary query values remain visible.
- `WithURLCredentials` and `WithClientURLCredentials` freeze metadata at option
  construction. Reused options compose immutable policies; later mutation of the
  supplied field slices cannot redirect or remove a mask.
- The shared HTTP example-server base selects `Fields()` only for services hosted
  by the current server, across HTTP and JSON-RPC. JSON-RPC inherits this section.
  New generated examples configure access logging; existing create-only example
  files require the documented wiring update.
- No transport/schema changes, credential policy duplication, or new dependency.

Selected evidence uses Go 1.27.2 on darwin/arm64 and explicit local source mode.
The fixture module pins the pushed `ee4552dc94ee` source for remote reproduction;
parent and candidate local replacements select their actual implementation roots.

1. Direct runtime tests cover 28 URL/consumer combinations: conventional key and
   camel-case names, userinfo, repeated query values, escaped and custom query
   credentials, and credential paths across server, client success, transport
   error and HTTP error status. Eight additional requests cover immutable option
   construction, two-option composition, reuse and malformed-query omission.
   All assert request preservation and appropriate ordinary-value visibility.
2. Generated runtime checks compile/vet the complete multi-host fixture module and
   run each exact emitted server logging configuration with its generated metadata
   under `go test -race`. Mixed, HTTP-only, JSON-RPC-only and ordinary hosts preserve
   request URLs, redact their own credential locations, and retain ordinary or
   unrelated-service query values. The JWT path method explicitly opts out of
   OpenAPI because that credential location is not an OpenAPI security contract.
3. Isolated parent/candidate/repeat generation covers the existing mixed diagnostic
   fixture, ordinary/disabled security controls, and the new four-host contract.
   All three modules build and vet at both parent and candidate. Of 138 artifacts,
   exactly four change: credential metadata imports and the `log.HTTP` option in
   `existing/cmd/diagnostics/jsonrpc.go`, `hosts/cmd/mixed/jsonrpc.go`,
   `hosts/cmd/webonly/http.go`, and `hosts/cmd/rpconly/jsonrpc.go`. The other 134,
   including all transport/schema outputs and ordinary server examples, are exact
   matches. Candidate and independent-process repeat match byte for byte.
4. Existing HTTP example goldens and diagnostic projection seams pass unchanged.
   Full `clue/log` and `observability/diagnostics` race suites pass. Focused lint
   reports zero issues; documentation checks pass.
5. TLC 1.7.4 on Corretto 25.0.4.1 reproduces `NoCredentialDisclosure` failures for
   NamesOnly (17 states), LogicalName (17), and Borrowed (26). Snapshot checks all
   30 states successfully. The model is unchanged; correspondence documentation
   now identifies the access-log consumer and generated host-selection tests.

Evidence receipts: `/tmp/loom-721-artifact-hashes.json`,
`/tmp/loom-721-artifact.diff`, `/tmp/loom-721-final-{artifacts,builds}-{parent,candidate}.log`,
`/tmp/loom-721-final-artifacts-repeat.log`, `/tmp/loom-721-generated-jwt.log`,
`/tmp/loom-721-runtime-race.log`, `/tmp/loom-721-direct-final.log`,
`/tmp/loom-721-generator-seams.log`, `/tmp/loom-721-lint.log`,
`/tmp/loom-721-docs.log`, and `/tmp/loom-721-model-*.log`.

An initial Basic path variant reproduced the separate known #842 CLI duplicate
parameter failure at the parent. Its exact input and failing build are retained
in `/tmp/loom-721-basic-path-repro.go` and `/tmp/loom-721-basic-parent-build.log` for
that issue. The final JWT fixture avoids this unrelated defect and uses full
module compilation; no acceptance gate is weakened.

Independent final review, integration, commit/push and ticket closure remain root
orchestration steps. Temporary outputs remain available until review completes,
then must be reduced to compact evidence and removed after checking for active users.
