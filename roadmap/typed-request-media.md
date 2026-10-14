# Typed request media — #787

Accepted user contract: the built-in request decoder recognizes valid +json media
types everywhere. Generic JSON decoding does not implement media-specific business
semantics. Existing unknown-media rejection, custom codecs and body limits remain.

1. Preserve original importer failure and add direct request-selector and DSL
   acceptance/rejection regressions before production changes. Cover ordinary JSON,
   vendor/parameterized JSON, malformed MIME, incompatible codecs, effective mapped
   Content-Type conflicts, absent body and excluded framing/raw/RPC/WebSocket scopes.
2. Extend existing ContentType at method HTTP request scope. Expression admission
   resolves typed JSON media using the shared httpcodec selector and effective body
   owner. Carry the admitted authored media through request IR, never infer it
   independently in the importer, renderer or OpenAPI builder.
3. Imported single JSON representations emit their exact noncanonical media.
   Generated clients set this media only within the existing supplied-body branch,
   before invoking an injected encoder. OpenAPI consumes the same request identity.
   Ordinary application/json remains implicit and generates identical bytes.
4. Reuse bounded nontrivial specimens for actual importer→generation→mounted server
   checks. Cover required/optional, absent/null/empty/supplied body, exact media and
   original header visibility for custom codecs. Preserve error matrix empty,
   whitespace, malformed, truncated and valid JSON, service invocation and payload
   state; verify unsupported MIME and limits at runtime and generated seams.
5. Compare parent/candidate representative outputs and compile modules; repeat
   generation in isolated processes, parse specs with libopenapi and pinned Vacuum.
   Use existing models only if changed representation/publication ordering needs
   a new proof obligation; no concurrency or independent state owner is introduced.
6. Document adopted semantics and regeneration. Freeze exact source/evidence for
   fresh independent review, atomic owning-ticket commit/push/closure, then cleanup.

Scope excludes general media negotiation, multiple-media choice, service/API
inheritance, new raw-body APIs or endpoint-specific runtime policy flags. Existing
response content types, dynamic-only header designs and form/multipart boundaries
keep their established owners.

Implementation status: accepted policy; parent regressions first. The request
ContentType declaration is method-local and retains exact authored MIME spelling.
No new lifecycle or ordering state is introduced: publication stays inside the
existing supplied-body branch, before the injected encoder factory. Existing body
presence and decoder error models remain applicable; direct/mounted tests check
this unchanged ordering boundary.

## Completed implementation and bounded evidence

- [x] Reproduced the importer loss and request-selector rejection on parent
  `19cf194ad`; added admission tests before changing production.
- [x] One endpoint declaration records exact authored media. Effective body and
  request wire bindings admit it before IR publication. Request IR supplies
  client encoding and OpenAPI; the importer request-mode owner uses the same
  codec classifier as runtime admission.
- [x] Valid concrete `+json` MIME labels now decode as JSON globally. Missing media
  remains JSON; malformed, unknown, wildcard and non-JSON suffixes remain rejected.
- [x] Reused alias and nullable request-body harnesses for required/optional,
  absence/null/empty, normal/error payloads, exact status/code/detail and service
  state. Custom encoder factories see media before encoding; custom decoder
  factories see original headers. Imported supported-fixture client/server wire
  checks also cover unknown/malformed media and configured body limits.
- [x] Direct expr/DSL/transport IR/shared codec package race checks pass. Supported
  form/multipart/raw-media importer controls remain unchanged and pass.
- [x] Exact archived-parent and candidate CLI generations of the same supported
  fixture, ordinary and vendor variants, compile and vet. Candidate artifacts
  repeat byte-for-byte in separate CLI processes. Four JSON/YAML outputs pass
  pinned Vacuum; the imported vendor contract also passes libopenapi parsing.
- [ ] Independent final review, integration, atomic delivery and cleanup.

Intended output changes: ordinary generated Go/spec bytes stay identical. Its
`gen/loom.json` design digest changes because the existing fingerprint includes
all exported expression fields, including the new absent media field. Vendor
output additionally changes one client encoder header assignment and the JSON
and YAML request content keys. No fingerprint exception or output normalization
is added to hide the representation change.

Comparison commands initially used different executable basenames, which changed
command comments, and a `/tmp` overlay path did not select the parent probe.
Those attempts are excluded. Accepted comparisons use canonical overlay paths,
actual archived parent source, and the same `loom` command basename; no CLI
subprocess depends on an overlay for its generator implementation.

Evidence packet: `/tmp/loom-787-final-manifest.json`; parent direct failure log,
focused direct/runtime/generated checks, actual artifact hash maps and exact
repeat comparison are retained there. The JSON decode error matrix and body
presence models are reused unchanged; no concurrent state or request negotiation
state machine was introduced.

Checked-in fixture scope: both HTTP and JSON-RPC ticktock were regenerated from
actual parent and candidate sources in temporary copies. All generated Go/spec
bytes match; only `loom.json` changes. Both copies compile and vet. The parent
JSON-RPC full fixture test has an unrelated pre-existing expectation mismatch:
`TestTypedServiceStreamContract/protocol-error/{plain,intercepted}` expects
`JSON-RPC error -32603:` while runtime reports
`jsonrpc: code -32603: internal server error`. This is preserved separately in
`/tmp/loom-787-parent-fixture-assertion.json`; it was not repaired or waived as a
successful full-suite result here. HTTP's parent full fixture tests pass.
