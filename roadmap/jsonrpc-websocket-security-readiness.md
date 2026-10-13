# JSON-RPC WebSocket security admission — #728

Status: implementation complete; frozen for fresh independent final review.
Parent: `abeb200e9f200ec93d330a2040e5f30b1dba5737`.
Managed worktree: `/Users/luca/.codex/worktrees/jsonrpc-websocket-security/loom`.

## Contract and owner

Existing JSON-RPC WebSocket admission forbids header, cookie and parameter
mappings. The shared `expr.httpWireBindingPlan` resolves both authored mappings
and effective credential locations. Service validation currently checks raw
mapping objects before endpoint validation calls `ensureWireBindings`; Basic
credentials never appear in the header mapping object. Both paths can evade
admission. Authentication scope selection remains owned by `MethodExpr`.

This investigation must distinguish explicit body credentials from unsupported
handshake mappings. It must not reject supported message credentials merely
because they are Security, introduce handshake authentication, or silently clear
requirements. NoSecurity and anonymous overrides retain their accepted meaning.

## Sequence

1. [x] Read #728, framework instructions and shared auth/binding models; create
   managed worktree from main.
2. [x] Add complete DSL failure/normal controls across scheme, stream and scope;
   establish current generated failure rather than assume the old panic persists.
3. [x] Model the actual ordering/representation omission; compare effective-plan
   admission with raw-mapping and blanket-auth alternatives.
4. [x] Record bounded generation/compile/runtime obligations and actual evidence.
5. [x] Independent readiness review before production edits.
6. [x] Implement at the shared effective-binding admission boundary.
7. [x] Run affected owner and representative generated/output checks; compare
   parent/candidate and independent repeat bytes. Update public docs and guides.
8. [ ] Independent exact final review, atomic integration/commit/push/closure by
   orchestrator, then remove inactive temporary artifacts after compact receipts.

## Bounded acceptance

Unsupported effective request bindings fail during DSL validation without
finalizer/codegen panic. JWT, API key, OAuth, Basic and session requirements
must keep all effective roles and scope; no implicit binding may evade the
same rule applied to an explicit mapping. Keep ordinary HTTP WebSocket,
JSON-RPC HTTP/SSE, anonymous and NoSecurity controls. Explicit body credentials are preserved when their source and security roles
survive normalization, as established by the mounted controls below.


## Findings and intended repair

- **Medium — current #728 admission gap:** 26 complete DSL cases accept effective
  header/cookie credentials although the existing service policy forbids them.
  Eight scheme/alternative kinds × server/client/bidirectional, plus JWT API and
  method inheritance, reproduce the omission. Explicit header/cookie/query
  mappings still reject. Original JWT bidirectional generation still panics:
  `build HTTP field init for v: token is a string but v. type is object`.
  The original server case now builds; do not claim its old finalization panic
  still reproduces.
- **High — silent credential loss at the same boundary:** explicit body JWT,
  API-key and OAuth designs with a distinct streaming payload all pass admission.
  Nine cases cover absent members, an existing member without its security tag,
  and client/bidirectional streams. `finalizeJSONRPCBodyState` changes the method
  Payload after method validation; `buildCredentialSchemeData` returns nil when
  the final payload lacks its tag. Generated JWT body-bidirectional endpoints
  have no authorizer calls. A mounted server accepts invalid and absent tokens,
  invokes the service three times and invokes authentication zero times. This
  is a reproduced bypass, not just a generated-code expectation.
- **Supported control:** explicit body JWT/API-key/OAuth server methods retain
  authentication. JWT bidirectional with corresponding tagged StreamingPayload
  also retains it. Each mounted control rejects invalid/absent credentials and
  accepts the valid credential (three authorizer calls, one service call).
  Rejecting all Security would break an existing working contract.

One effective-admission boundary must establish both physical location and
survival of every required credential role before finalization. Use the existing
wire plan for actual locations and scheme/source membership, including Basic;
do not infer credentials from header objects or duplicate scheme location
selection. Extract the JSON-RPC effective request-source choice as one owner
consumed by validation and normalization, so the admission view cannot diverge
from the source later passed to service analysis. Check each effective role
against that selected request source; matching field spelling without the
credential tag is insufficient. Preserve authenticated alternatives even beside
an anonymous alternative. Do not repair this by deleting requirements, by
restoring missing auth in individual generated handlers, or by authorizing the
connection handshake.

Source/role incompatibility must produce a design diagnostic identifying the
method and credential, rather than silently copying credential tags into a
separately authored StreamingPayload. The working body controls use matching
credential names and roles. Renamed or incompatible source mappings must be
examined against the accepted source contract during implementation; this
readiness does not authorize a new credential mapping syntax or an arbitrary
constraint-equivalence policy. The final validation contract must reject all
cases that would lose an effective auth requirement and preserve ordinary
HTTP/SSE and current working message-body auth.

Keep service transport consistency and existing explicit mapping diagnostics.
No #752 per-scheme scope work, general streaming payload rewrite, new public
handshake feature, or broad credential-schema refactor is included. Public docs
must explain supported message credentials versus unsupported handshake binding;
previously admitted unsupported/bypassing designs will fail earlier and must fix
their design before regeneration.

## Readiness evidence

Go 1.27.2, Java 25.0.4.1, TLA+ pinned 1.7.4 (TLC reports 2.19).
At readiness, production files were unchanged; only fixture/test/model/plan files were added.

| Check | Result |
| --- | --- |
| Complete DSL red tests | 35 expected failures: 26 effective-location + 9 lost-role cases; `/tmp/loom-728-admission-red.log` |
| Explicit rejections and supported controls | 17 subcases pass; `go test ./expr -run '^TestJSONRPCWebSocket(ExplicitSecurityMappingRejection\|SecurityAdmissionControls\|MessageCredentialControls)$' -count=1`; `/tmp/loom-728-controls.log` |
| Current generation probe | 11 of 12 cases build/vet; implicit JWT bidi panics. The matching tagged bidi case was run separately; `/tmp/loom-728-generation.log`, `/tmp/loom-728-bidi-same.log` |
| Mounted message auth | Four generated modules build/vet and pass race tests; `TestJSONRPCWebSocketMessageSecurity`; `/tmp/loom-728-message-controls.log` |
| Scoped lint | `golangci-lint run ./expr/... ./jsonrpc/codegen/...` exits 0; `/tmp/loom-728-readiness-lint.log` |
| Actual bypass | `/tmp/loom-728-body-jwt-bidi-wire.log`: invalid and absent token both return accepted, auth calls 0, service calls 3 |
| Raw-mapping model | expected NoUnsupportedWireBinding failure, 115 states |
| Resolved-mappings model | expected Basic NoUnsupportedWireBinding failure, 118 states |
| Location-only model | expected NoLostCredential failure, 167 states |
| Blanket-security model | expected PreserveSupported failure, 124 states |
| Checked model | all three invariants pass, 168 states |

The model runs cover 42 bounded inputs, not arbitrary schema or generated-code
correctness. Model logs are `/tmp/loom-728-model-*.log`; model state is temporary.
The fixture and models preserve the reproduction after temporary cleanup.

## Reproduce generated evidence

The durable probe is `expr/tla/jsonrpc_ws_admission/generation_probe.go.txt`.
Make an overlay mapping an otherwise nonexistent
`jsonrpc/codegen/loom_728_generation_probe_test.go` to that source, then run:

```sh
LOOM_DIR="$PWD" LOOM_728_OUTPUT=/tmp/loom-728-generation go test -overlay=/tmp/loom-728-generation-overlay.json ./jsonrpc/codegen -run '^TestJSONRPCWebSocketSecurityGenerationProbe$' -count=1 -v
```

At the parent, the full probe fails at `jwt-bidi`; other subtests continue.
Copy `message_auth_harness.go.txt` to `auth_test.go` at the generated
`body-jwt-bidi` module root, not inside gen, and run `go test -race ./...` there.
It must fail with the recorded bypass. The same harness passes in the three
`body-*-server` modules and `body-jwt-bidi-same`. The durable positive generated
test already checks those four cases, including requirement-to-projection role
counts and exact scheme identities, before build/vet and mounted wire checks.

## Implementation verification obligations

1. Turn all 35 reds green and retain explicit rejection diagnostics; verify named
   credential payloads, partial role loss inside AND/OR alternatives and member
   type/tag mismatch at the same owner. Do not settle for field-name presence.
2. Rerun direct expr/auth/wire-owner tests and the four mounted message controls;
   add source-ownership/idempotency tests around the extracted effective selector.
3. Use the same fixture/renderer in parent, candidate and an independent repeated
   process. Compare every generated artifact byte for byte for supported body
   JWT/API-key/OAuth and tagged bidi, unsecured/NoSecurity WS, and POST/SSE JWT.
   Use existing HTTP WS fixture for the unaffected transport boundary. Build/vet
   all distinct accepted outputs. Intended supported-output differences: none.
4. Rejected implicit JWT/Basic/session and lost body-role designs must stop at
   validation with no generated artifacts, panic or silently weakened auth.
5. Only broaden checks if changed owner effects or failures justify it. No full
   exported-design corpus is a ticket acceptance requirement. Root coordinates
   public docs/shared guide changes and final exact integration review.

Root takeover: `/root/review_uniform_json_corrections` independently approved
manifest `5921efe94177a0e07915c5cd01e34520a702eac26029a699a6eb2588040f2a98`,
confirming all red/control cases, mounted auth behavior and model outcomes. An
implementation-agent service failure left production unchanged. Root follows the
approved test-first/model-first sequence, reuses frozen readiness evidence, adds
source/role/type edge regressions before production, then validates bounded
accepted/rejected outputs. A fresh nonimplementer must review the final diff
before any commit. Shared guidance and main integration remain root-owned.

## Final implementation receipt

The effective binding plan retains ordered credential source/tag roles when
physical operations merge. WebSocket admission reads that plan after resolution,
rejects non-body locations, and checks every role against the same effective
message-source selector used by finalization and request-ID validation. Basic
physical binding construction is a private helper; no location policy is copied
into generated transports. Existing raw mapping diagnostics remain intact.

All 42 admission regressions now pass: 35 readiness cases and seven role/type
edges (renamed, non-string, nullable, custom Go type, partial AND/OR loss and a
lost role sharing the same physical source). The extra six original edge failures
are preserved in `/tmp/loom-728-role-red.log`; the shared-source failure is in
`/tmp/loom-728-shared-role-red.log`. Named string/request controls pass. Selector
identity survives repeated normalization, and merged roles remain ordered and
unique. The prior implicit-header streaming-source test now uses a supported
explicit body mapping; unsupported implicit-header behavior has direct rejected
coverage. No expected auth requirement is removed to make the tests pass.

Acceptance with Go 1.27.2 and pinned golangci-lint 2.14.0:

- Complete expr package passes (`/tmp/loom-728-expr.log`, 0.949s).
- Four generated JWT/API-key/OAuth/tagged-bidi modules build/vet and pass mounted
  race tests: invalid/absent credentials reject, valid credentials invoke the
  service; `/tmp/loom-728-message-green.log`, 13.495s.
- Eight supported designs produce 95 artifacts identical between exact parent,
  candidate and independent repeat. All 24 modules build/vet. The comparison
  includes the four body-auth cases, anonymous/NoSecurity, POST and SSE.
  `/tmp/loom-728-comparison/hashes.json` records exact file hashes; `run.py`
  preserves the bounded command and overlays. Temporary modules pin pushed
  `v1.10.0-alpha.9.0.20261013010152-abeb200e9f20` with explicit local source.
- Four formerly bad generated designs now stop at DSL validation: implicit JWT
  server/bidi, Basic server and lost-role body bidi. The expected probe errors
  are in `/tmp/loom-728-rejected.log`; no output directories are created.
- Existing ordinary HTTP WebSocket/session OpenAPI and generated-shape controls
  pass (`/tmp/loom-728-http-ws-control.log`, 0.673s).
- Pinned expr/DSL/JSON-RPC lint reports zero issues
  (`/tmp/loom-728-final-lint.log`). Model sources/configurations are unchanged;
  reuse the independently checked 168-state positive and four negative results.

Public DSL documentation and consumer guidance explain per-message body auth,
unsupported handshake mappings and required preservation in a separate streaming
payload. Framework guidance/model correspondence identify the shared owner.
No supported generated bytes change. Final independent review must confirm this
exact diff before integration. Root owns delivery and inactive-artifact cleanup.

Fixture lint follow-up: explicit `./expr/testdata` lint exposed complexity in
`JSONRPCWebSocketSecurityDSL`; nested testdata packages are not selected by
`./expr/...`. Three private fixture helpers now separate security declarations,
payload credentials and transport mappings without changing evaluation order.
Complete expr tests and explicit fixture/expr/DSL/JSON-RPC lint pass. Two new
independent generations (`candidate-fixture`, `repeat-fixture`) retain the same
95 artifact bytes as parent and the prior candidate. Identical generated output
reuses the mounted auth evidence. The final re-review must cover this mechanical
fixture delta; no production code changed after the first review approval.
