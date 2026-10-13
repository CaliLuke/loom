# Error-property representation — #717

A full built-in problem response must preserve a ServiceError's Temporary,
Timeout and Fault properties independently of HTTP status or JSON-RPC code, as
the existing gRPC representation does. The shared public problem representation
currently omits all three properties, then its client adapter guesses them from
status. Fix this representation owner and derive both transports from it.

1. Read the runtime problem document, built-in ErrorResult shape, service error
   construction and shared HTTP/JSON-RPC projection. Reproduce all property
   combinations with misleading HTTP statuses and negative JSON-RPC codes before
   changing production; retain message/remedy/ID and gRPC controls.
2. Establish a bounded generated HTTP/JSON-RPC specimen and existing selected-body
   controls. Inspect actual consuming designs and docs for full/pruned/mapped
   ErrorResult usage. Record precise wire/body-selection semantics and any truly
   unsettled public decision before implementation.
3. Model loss/invention of error properties across service→public document→body
   projection→client, distinguishing explicit selected-body omissions from full
   problem responses. Use counterexamples to check that removing status inference
   and carrying properties eliminates both directions of corruption.
4. Propose one corrected public problem representation and client conversion
   owner; no per-transport flag inference. Independently review readiness before
   implementation. Prefer carrying properties explicitly rather than teaching
   clients another status/code classification table.
5. Implement only the approved owner change, update public/contributor guidance,
   regenerate affected checked-in fixtures and compare a bounded full/selected/
   mapped/custom error set at parent/current/repeat. Compile/vet, inspect raw wire
   values and client error fields, and validate rendered OpenAPI with pinned
   libopenapi/Vacuum. Keep unrelated error classification defects separate.
6. Freeze exact diff and evidence for fresh final review, then atomic commit/push
   and ticket closure. No full exported-design sweep is a ticket acceptance gate.

Initial source facts: runtime ProblemResponse and expr.ErrorResult omit the
properties; NewProblemResponse already resolves a ServiceError, so their values
are available before projection. Shared buildProblemClientResult reconstructs a
ProblemResponse but expands five arguments into ProblemErrorFromBody, which
infers flags from the transport status. Generated service constructors already
retain authored flags, while gRPC explicitly serializes/restores them.
Explicit Body projections are supported and must retain their authored omissions;
the representation repair must not silently add omitted fields to custom bodies.
Readiness was independently approved before implementation; no unresolved public
policy remains. The following receipt records the implemented candidate.

## Readiness proposal

The documented error-properties contract already promises these three properties
to clients; `docs/error-handling.md` even lists them in ErrorResult. Carry them as
required boolean fields `timeout`, `temporary` and `fault` in the full built-in
problem shape and runtime document, including explicit false values. The server
copies the resolved ServiceError values. The shared client conversion consumes
the reconstructed ProblemResponse directly; replace the five-argument
ProblemErrorFromBody API with a document argument and delete inferProblemFlags.
Generated clients and servers must be regenerated together. Existing handwritten
calls must supply a document and explicit properties. No old status-inference
mode or compatibility adapter is proposed.

Authored Body selection remains authoritative. Selected/pruned/empty bodies do
not gain omitted fields. Properties explicitly mapped to headers/cookies are
reconstructed through the existing shared mapping owner. An omitted property is
false in the reconstructed ServiceError; a status/code never invents it. For
example, full Timeout=true/Temporary=false mapped to 504 retains those values;
full all-false mapped to 500 stays all-false; an explicit pruned object
Body(func(){ Attribute("detail") }) intentionally conveys no properties.
Scalar Body("detail") and Body(Empty) automatically map the remaining members to
headers, so their reconstructed properties are retained. This follows existing selected-body behavior for omitted details,
IDs and retry hints. Undeclared response handling, safe error messages, joined
error classification and gRPC's representation stay outside this change.

The alternative of adding fields but retaining inference still corrupts values;
the model rejects it. Inferring only when fields are missing would preserve the
same corruption on explicitly selected bodies and would introduce a compatibility
policy without an authored contract. No unresolved product choice is identified.

## Parent evidence

- Parent `0d967158927e65dfc0274da97c7308a677a36c6e`, Go 1.27.2,
  GOEXPERIMENT=jsonv2, local source explicitly selected.
- `http/problem_properties_test.go` checks eight property combinations across
  five misleading HTTP/JSON-RPC status values. All 120 raw-field checks fail
  because the fields are missing: `/tmp/loom-717-parent-wire-red.log`.
- `codegen/generator/error_properties_test.go` generates and vets three actual
  transport adapters. Mounted HTTP and JSON-RPC produce 48 missing-wire-property
  failures plus lost/invented reconstructed flags. All eight gRPC property
  combinations pass through a generated client/server over bufconn. Name, ID,
  detail, invocation and actual status/code assertions pass. Final valid harness
  log `/tmp/loom-717-parent-generated-red.log`; earlier harness syntax,
  server-arity and case-sensitive JSON decode mistakes are not bug evidence.
- Existing `TestSelectedProblemBodyConstruction` owns 16 full, scalar-selected,
  shaped, empty, mapped-header/cookie, override and custom cases. Reuse and extend
  its wire harness for retained/omitted properties and malformed full booleans.
- The consuming registry design under `loom-mcp/registry/design` declares seven
  ErrorResult errors with independent gRPC code mappings. It is an unchanged
  consumer control, not an HTTP adoption claim. Repository default-error and
  selected-body fixtures supply the actual HTTP contract shapes.
- The bounded model under `http/tla/problem_properties` rejects current and
  wire-only designs at 777 states (PreserveSelected counterexample) and checks
  explicit transport over 1,024 states. Logs `/tmp/loom-717-model-*.log`.

## Implementation acceptance

1. Direct runtime tests establish actual booleans, identity and safe-message/
   retry-hint behavior; direct expr/schema assertions establish required fields.
   The shared client adapter must consume the whole public document with no
   status/code classification path.
2. Extend the generated three-transport specimen to use generated error helpers
   as well as explicit runtime values, proving DSL property construction too.
   Exercise HTTP and JSON-RPC full, scalar-selected, pruned and header/cookie
   projection paths. Confirm all-false and misleading statuses, and reject
   missing/null/malformed booleans in a full declared response. gRPC bytes and
   decoded flags remain unchanged.
3. Compare parent/current/repeat output for the three-transport specimen plus
   existing meaningful default/selected/custom error designs. Inspect every
   intended difference; compile/vet both parent and candidate modules. Repeated
   generation occurs in separate processes. Reuse selected-body tests, avoiding
   the full exported-design corpus.
4. Regenerate affected checked-in integration fixtures and inspect their exact
   diffs. Run affected error/generator tests, scoped lint and fixture parity.
   Parse full and selected rendered OpenAPI with libopenapi and lint them with
   the pinned Vacuum rules; generated booleans must match the document schema.
5. Update error docs, consumer/framework skills and model correspondence; obtain
   independent exact final review before one commit for #717. Keep the original
   dependency race #820 separate from model/schema correctness.

Implementation acceptance exposed and corrected the readiness example for scalar
and empty bodies: the existing mapping owner emits all remaining ErrorResult
members as response headers. Direct client initialization confirms assignment of
Timeout, Temporary and Fault from those headers. The existing selected-body test
already represented this behavior for details, ID and retry hint. Preserve this
mapping; only explicitly pruned object members are absent from the wire.

## Final candidate acceptance

Scope is the shared error-property representation for #717, on parent
`0d967158927e65dfc0274da97c7308a677a36c6e`. Production changes are limited to
`http.ProblemResponse`/construction/reconstruction, the built-in `expr.ErrorResult`
shape, and the shared HTTP/JSON-RPC client conversion call. The old status flag
inference is deleted. Other error classification and custom error shapes remain
outside this repair.

- Direct runtime covers all eight property combinations over five independent
  statuses/codes, exact false fields, safe detail, remedy and ID; omission never
  derives flags from status. Full HTTP, expr and JSON-RPC runtime packages pass
  (HTTP 4.004s; expr 1.183s; JSON-RPC 1.075s). Focused HTTP/expr race checks pass
  1.583s/1.578s. Existing JSON/XML and remedy expected bytes now explicitly include
  the three false fields.
- Generated three-transport checks pass 2.866s. Every DSL property combination
  travels through its generated error constructor, endpoint and client. HTTP and
  JSON-RPC reject each missing, null and string-valued boolean; gRPC remains a
  byte-identical control. Invocation, status/code, name, ID and detail are checked.
- The expanded 21-case selected-body matrix passes 29.706s, including scalar
  properties, pruned objects, empty bodies, custom error, overrides, header and
  cookie projections, safe messages, formatter behavior and malformed full-body
  booleans. This uses the existing public representation and mapping owners.
- Seven bounded designs generated at parent, candidate and process-isolated
  repeat produce 118 artifacts each. All 118 repeat exactly. Seventeen intended
  changes add fields/validation/construction to full bodies, add automatic header
  encoding/decoding and required-header contracts for scalar/empty bodies, and
  replace status-based helper calls with the reconstructed document. Pruned
  bodies change only that call; the custom design and every gRPC artifact are
  unchanged. All 14 parent/candidate modules build/vet (8.461s/7.500s). Exact maps
  and full diff: `/tmp/loom-717-comparison/{hashes.json,artifact.diff}`.
- Checked-in HTTP ticktock regeneration changes five of 18 artifacts: client and
  server types, JSON/YAML specs and evaluated-design digest. Parent regeneration
  exactly matches its checked-in tree. Candidate independent regeneration parity,
  compile and stream/pre-stream rejection checks pass 5.127s. The fixture's fake
  declared-error document now includes required false fields; its response-body
  ownership assertions remain intact. No manual generated-file edits.
- Five HTTP helper goldens and eight existing OpenAPI goldens are regenerated.
  Affected HTTP structural/golden suites pass 0.752s; four OpenAPI specimen pairs
  pass 0.419s. The meaningful `ProblemPropertiesDSL` extends the default-error
  fixture with pruned, scalar, mapped-header and custom error controls. JSON/YAML
  specs parse with libopenapi, pass pinned Vacuum rules and match independent
  subprocess generations (9.509s). No dependency version changed.
- Scoped golangci-lint v2.14.0 reports zero issues for HTTP runtime, expr, shared
  generator, HTTP generator/testdata and OpenAPI v3. Go 1.27.2 with jsonv2 and
  explicit local source is used throughout. Pushed module references remain in
  reusable generation harnesses. Models retain the independently reviewed
  current/wire-only counterexamples and the 1,024-state explicit-property check.

Public error guidance, both consumer/contributor skills and formal model index
are aligned. Atomic final review is pending; no delivery or overall CI success is
claimed. This acceptance uses a bounded owner/transport/specimen set, not the full
exported-design corpus.
