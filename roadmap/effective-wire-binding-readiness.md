# Effective HTTP wire bindings readiness

Status: independent readiness approved by `review_effective_repair`; production
implementation complete; bounded acceptance recorded below, pending final review.
Parent: `8c7d8d711b9e2fe54fbff1dabb9b49fd361ed170` (pushed).
Worktree: `/Users/luca/.codex/worktrees/effective-wire-bindings/loom`.
Candidate owning report: #756, including the proven #729 credential witnesses.
This is one shared mapping-owner repair, not separate transport patches.

## Recovery sequence

- [x] Reproduce explicit request/response collisions and retain valid controls.
- [x] Inspect inheritance, implicit auth/session/error mappings, body exclusions,
  and evaluation ordering.
- [x] Demonstrate actual generated request/response value loss and auth ambiguity.
- [x] Compare candidate admission rules with a finite model; reject blanket and
  explicit-only validation. Remove the unapproved required/default hypothesis.
- [x] Record a concrete shared owner, intended differences and bounded acceptance.
- [x] Obtain independent readiness review; no findings remained.
- [x] Implement the owner and replace the superseded inference paths.
- [x] Complete the bounded direct/output/compile/wire/model acceptance below.
- [ ] Freeze the exact diff, obtain independent final review, then let root commit,
  push, reconcile proven tickets, update shared guides and clean scratch evidence.

## Established failure and ownership

The reports describe independent values assigned one wire slot. Current generated
request headers overwrite the first value, query decoding selects the first
value for both fields, cookies decode both fields from one cookie, and response
headers overwrite the first result. These are design errors that must stop
before generation, with both field names in the diagnostic.

`MappedAttributeExpr` preserves forward field-to-wire mappings, but its reverse
map retains only one field per wire name. `Remap` normalizes authored keys, and
`Merge` applies nearest-scope remapping. The owner must enumerate forward mapped fields without using the reverse lookup; validating through the reverse map loses the witness.
Likewise, inspect the complete Params mapping before `PathParams` selects one
attribute for each route wildcard. Classify its wire names against the route
wildcard set instead of validating an already lossy derived object.

`HTTPEndpointExpr.Prepare` merges API/service/canonical/endpoint mappings before
validation. However, `MethodExpr.Validate` injects session payload fields, while
`HTTPEndpointExpr.finalizeRequirements` adds implicit auth/session mappings only
in finalization. Endpoint Prepare is too early for a complete plan, and checking
only explicit declarations during Validate misses late collisions.

There is a second semantic disagreement: `defaultRequestHeaderAttributes`
concatenates method/service/API security independently of selected requirements.
The direct scope witness selects a method API key but loses a different field
carrying the overridden service key from the inferred request body. Body
exclusion must consume the selected mapping owner too.

Responses and errors share physical slot rules. Response-cookie duplicates are
already rejected; this is a preservation control. Response headers lack that
check. Built-in ErrorResult fields can gain `loom-attribute-*` headers in
`mapUnmappedAttrs` after validation when the error body is Empty or selects one
field. Those inferred headers must enter the same plan before validation. A
normal full error body must not acquire these headers.

## Proposed representation and lifecycle

Add a package-private binding plan under `expr`, owned by each effective HTTP
endpoint and response/error occurrence. No new public DSL or mode is needed.
Each logical operation retains:

- physical slot: request/response occurrence, location and canonical wire name;
  headers fold case, query and cookie names preserve case, and path is separate;
- source identity: payload/result/error/transport-owned source plus an ordered
  field tuple; Basic is one `(username, password)` operation;
- wire codec: raw value, Basic frame or Bearer frame, selected using the same
  existing Authorization-versus-custom-location rule;
- effective requirement membership; the normalized detached maps retain the
  selected authored/inherited/inferred mapping after scope composition.

Source identity is the field occurrence, not its current value or a reused
AttributeExpr pointer. A field used by multiple schemes with the same physical
codec is one operation with combined membership. A field tagged for incompatible
codecs remains two operations and cannot disappear through map replacement.
An explicit credential mapping must be classified as the credential operation,
not counted a second time as an unrelated raw mapping. An ordinary field is
unconditional and coexists with every applicable auth requirement.

Build the complete request plan at the start of endpoint validation, after the
method validation pass has injected session fields and after preparation has
merged scopes. Use `EffectiveRequirements` / effective sessions to select the
same nearest-scope auth contract as the method; preserve detached ownership.
Resolve explicit and inferred mappings once, including Basic's virtual composite
and transport-owned session cookies. Publish normalized detached mapping and
requirement projections before existing type/body validation needs them. Keep
the validated semantic plan through finalization; finalization may initialize
finalized types but must not independently discover new mappings or reselect
security. Remove the superseded late auth/session selection and the independent
body-exclusion scope walk. Body inference consumes the plan's source exclusions.

Plan each response/error occurrence independently. Move the implicit built-in
problem-header selection into this owner using the authored body selection;
retain the existing Empty/single-field/full-body rules. Response finalization
uses its validated projection. Do not compare headers across separate response
alternatives: response accumulation and OpenAPI merging have their own owner.

Implementation must confirm phase correspondence, including malformed designs
that already have method errors, and JSON-RPC streaming normalization that moves
payload state after validation. Semantic source IDs must survive that move; do
not introduce a second fallback inference engine for direct helper callers.

## Admission contract

Within each physical slot, coalesce exact source-tuple/codec reuse. For every
remaining pair, reject unless both are mutually exclusive auth alternatives,
use disjoint source fields, and have disjoint known framing domains. Basic and
Bearer are disjoint; raw API-key strings overlap both and independent Bearer
credentials share the same domain. A common AND requirement cannot carry two
independent values in one slot.

| Case | Proposed result |
| --- | --- |
| `a:X-Name`, `b:x-name` request or response headers | Reject, naming both fields and the slot |
| Distinct attributes with the same query/cookie/path wire name | Reject |
| Query `Name` and `name`, cookie `Name` and `name`, or same name in separate locations | Accept |
| Inherited `a:X-Old`, endpoint remaps `a:X-New` and maps `b:X-Old` | Accept; nearer remap removes the old occupancy |
| JWT and raw API key both default to Authorization, AND or OR | Reject; map one credential elsewhere |
| Basic OR Bearer with separate fields, implicit or explicit Authorization | Accept, preserving the delivered #732 contract |
| Same source field checked as JWT and OAuth with the same Bearer codec | Accept, including an AND requirement |
| Separate JWT and OAuth fields both using Authorization | Reject; framing cannot select the source |
| Basic username reused as Bearer source, or same source tagged raw API key and JWT in Authorization | Reject incompatible representations |
| Ordinary field collides with an inferred session cookie or error header | Reject |

Diagnostics must identify both sources, physical location/name and relevant
schemes; suggest a distinct explicit mapping where that resolves the conflict.
Ordering and diagnostics must be deterministic. Preserve existing response-cookie
rejection and same-source inherited remapping precedence. Do not mutate ancestor
mappings or the selected method requirements while planning an occurrence.

This corrects #729's blanket Basic/Bearer suggestion in light of delivered #732.
No required/default admission policy is added. Requiredness still constrains the
payload under #732. Generated probes also record that defaulted Basic credentials
can be always-send and replace Bearer, and that supplying both alternative
credentials cannot preserve both in one slot. These are separate M7 presence and
selection obligations; this repair must not claim to fix them or make those
designs newly invalid. The model intentionally excludes that policy question.

## Red evidence and model

The original parent `2e4c80226ad2f84c27d506558f34e961344bc908` produced seven red
cases and six controls in `/tmp/loom-wire-binding-readiness-red.log`. The managed
worktree was safely fast-forwarded to pushed `8c7d8d71`; all then-existing readiness
file hashes were preserved in `/tmp/loom-wire-binding-prefastforward.json`.

At the new parent with Go 1.27.2:

- `go test ./expr -run TestHTTPWireBinding -count=1 -v`: 17 failing leaf cases,
  eight passing controls. The scope/body omission, inherited/path collisions,
  both session-cookie ownership modes and inferred error headers are included.
- `LOOM_DIR="$PWD" go test ./http/codegen -run '^TestHTTPWireBindingGeneratedReadiness$' -count=1 -v`:
  eight failing generated request cases and one failing response case. Optional
  Basic/Bearer branches and shared-source/same-Bearer round trips pass. The
  required/default exploratory cases record observations without claiming fixes.
- The durable `expr/tla/http_wire_bindings/shared_source_probe.go.txt`, copied
  temporarily into the HTTP codegen tests, fails full generated compilation for
  duplicate CLI parameters/locals when Basic username and Bearer reuse a field.
  That is representation conflict evidence, not permission for a naming patch.

Generated probes use the verified pushed module version
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` with an explicit local replacement
for this worktree. Logs are `/tmp/loom-wire-binding-expanded-red.log`,
`/tmp/loom-wire-binding-generated-red.log` and
`/tmp/loom-wire-binding-shared-source-red.log`. The generated probe is a parent
reproduction; after admission is implemented its invalid designs must be split
into direct rejection tests and separately generated valid controls.

The model and run instructions live at `expr/tla/http_wire_bindings/README.md`.
With the pinned JAR, `checked` explores 7,296 states and passes. `legacy` fails
`NoAmbiguousDecode` after 3,657 states; `blanket` fails
`PreserveDisjointAlternatives` after 3,793; `explicit-only` fails
`NoAmbiguousDecode` after 3,659. Counterexamples drove source-tuple/codec identity
and complete implicit-binding admission. This bounded pairwise model does not
prove extraction, phase ordering, HTTP parsing, generated output or public intent.

## Bounded implementation acceptance

1. Convert the direct readiness cases into durable table-driven owner tests,
   adding before/after finalization equality, detached inheritance, diagnostic
   determinism, ordinary Authorization versus Basic, shared AND/OR membership,
   error full-body control, and JSON-RPC streaming source preservation. Invalid
   or missing credential fields must return design errors without panics.
2. Compare exact output at the stated parent and candidate using representative
   existing HTTP auth, parameter/header/cookie, response/error, session and
   HTTP/JSON-RPC streaming designs. Extend existing fixtures where needed. Cover
   implicit/explicit Basic OR Bearer, required/default controls, same-source
   same-codec reuse, distinct-location JWT+key, overridden security/body fields,
   inherited remapping and inferred error headers. Rejected witnesses must fail
   before rendering; record their parent evidence rather than bypass validation.
3. Intended differences: deterministic rejection diagnostics for ambiguous
   designs; explicit Basic/Bearer Authorization alternatives become valid;
   overridden-auth fields remain in inferred bodies. Valid distinct mappings and
   supported implicit Basic/Bearer output should remain byte-identical unless
   normalization removes demonstrably duplicate operations. List every actual
   difference and explain it before review; do not silently update broad goldens.
4. Compile and vet the full generated client/server/CLI modules for the selected
   valid HTTP and JSON-RPC designs, with verified pushed module pins and explicit
   local source. Run real encode/decode and auth-branch checks for distinct-slot
   JWT+key, both Basic/Bearer alternatives, shared-codec reuse and scope/body
   preservation. Include wire values, decoded payload and service/auth invocation
   where endpoint behavior is relevant. Repeat generation in separate processes
   and compare bytes; do not repeat identical builds just for determinism.
5. Check OpenAPI mapping and auth requirements from the same evaluated design;
   parse/lint affected rendered specimens with existing pinned tooling. Keep
   response accumulation semantics unchanged. Run affected expr/HTTP/JSON-RPC
   packages and existing #732/#794 regressions. No full exported-design audit.
6. Rerun the aligned model and freeze the final diff/evidence for fresh review.
   Root owns the atomic delivery, issue disposition, public/shared-guide edits
   and cleanup. Ticket closure is limited to reproduced and corrected witnesses.


## Implementation and acceptance record

The private owner is `expr/http_wire_bindings.go`. Request validation resolves
selected requirements and detached header/query/cookie projections once. Body
inference consumes that plan; finalization no longer independently inserts auth
or session mappings. Response/error validation sees inferred problem headers
before checking occupancy. Existing response-cookie diagnostics remain intact.
JSON-RPC streaming finalization now uses the pre-normalization source plan;
the parent panicked after moving payload state. Direct tests cover that boundary.

With Go 1.27.2 and explicit `LOOM_DIR`, the direct expr/DSL suites, owner tests,
HTTP/JSON-RPC auth preservation tests (#732/#794), generated full CLI build/vet,
race-enabled wire/auth/scope harnesses and scoped lint passed. The aligned model
reproduces all three expected failures and the 7,296-state checked success.
Rendered mapping/scope and OpenAPI 3.2 specimens use libopenapi and pinned Vacuum.

The bounded comparison uses pushed parent `8c7d8d71` and the pinned module version
above with explicit local source replacements. Eleven representative designs
produce 167 artifacts: HTTP mappings, credential defaults, custom errors, selected
scope/body, sessions, streaming sessions, built-in full/empty/mapped errors and
the OpenAPI 3.2 specimen; JSON-RPC mappings, scope/body and sessions. Every selected
parent/candidate generation succeeds. All eleven candidate full client/server/CLI
modules build and vet; the changed parent modules do too. Independent processes
produce identical candidate bytes. No checked-in generated fixture changes.

Exact intended output differences:

- HTTP scope/body: eight artifacts restore the ordinary `other` field that was
  incorrectly excluded by overridden service security, including CLI flags,
  client/server body conversion, request-body schema and body requiredness.
- JSON-RPC scope/body: six client/server/CLI artifacts restore that same field;
  its OpenAPI already retained the correct body and stays unchanged.
- `OpenAPI32FeaturesDSL` previously used Basic AND OAuth in one Authorization
  header. Its specimen now uses Basic OR OAuth, keeping both external-security
  URI and device-flow feature coverage. Five generated artifacts change:
  service security documentation, endpoint auth selection, optional Bearer
  framing, and JSON/YAML security requirement alternatives.
- Every other compared artifact is byte-identical. Explicit uppercase
  Authorization Basic/Bearer alternatives now validate and compile; ambiguous
  designs are rejected before output rather than silently overwriting values.

The JSON-RPC full codegen package passed. A full HTTP package run reached its
cumulative ten-minute deadline; it is not reported as passing. Its optional-body
text assertions also fail at the parent and were independently fixed on main in
`589e96b1`. The invalid OpenAPI fixture was corrected here; its focused test
passes. An isolated type-identity WebSocket test and the final viewed-WebSocket
test both pass after one broad-run close race. No full-corpus rerun was needed.

An additional selected built-in ErrorResult body (`Body("detail")`) has identical
parent/candidate output but fails full compilation (client body/header local
collisions and a ServiceError.Detail selector). Its durable `selected_error_body_probe.go.txt` and both
outputs are preserved for separate triage. Selected-body expr ownership remains
covered; the valid compiled built-in error fixture covers full/empty/mapped
bodies. This unrelated generator defect is not claimed fixed by #756.

Review evidence lives under `/tmp/loom-wire-compare`, `/tmp/loom-wire-extras`, and
`/tmp/loom-wire-binding-*.log`; a final content-hash manifest identifies exact
source and logs. Remove scratch only after independent review and delivery.

## User-facing migration text for root integration

Each HTTP header, query parameter, cookie or path parameter must identify one
logical value. Header names compare without case; query and cookie names retain
case. Loom now rejects conflicting explicit, inherited or inferred mappings
before generation, naming the affected fields and wire slot. Give independent
values distinct wire names. Basic and Bearer authentication may share
Authorization as alternatives when they use separate credential fields; raw API
keys or independent Bearer credentials require distinct mappings. Exact reuse
of one source with the same framing is valid. Regenerate clients and servers;
fields belonging to overridden security declarations remain ordinary body fields
unless the selected method mapping puts them elsewhere. Required/default and
explicit-empty policies are unchanged by this repair.

## Integration review boundary

The isolated 27-file implementation passed fresh independent review by
`/root/review_wire_bindings_final` without findings. Manifest SHA:
`71f7c96021c67e5919ac05b212a294ec98673c2e8eb3a28620f40953f532fadd`;
exact diff SHA `706ea2850465d83891a8d7ab78a85fcb68e9f2c4e5f46b114eee2e1ba4cd962d`.
The reviewer independently checked 501 parent/candidate/repeat artifact hashes,
expr/DSL and generated HTTP/JSON-RPC gates plus additional mapping probes.

Root integrated the source files byte for byte at `dc88a33c2`, with no overlapping
source changes. Public DSL and consumer guidance now explain rejected collisions,
valid Basic/Bearer alternatives and migration. Framework guidance and the model
index identify the shared owner. Integrated focused expr, HTTP, JSON-RPC and
OpenAPI tests pass (0.560s, 10.826s, 9.680s and 0.702s). Final exact integrated
diff review remains the commit boundary. #756 is the atomic owner repair; its
explicitly subsumed credential case #729 is reconciled from the same evidence.
