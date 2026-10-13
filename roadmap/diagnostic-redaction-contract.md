# Diagnostic redaction ownership — #718

Parent: `0d967158927e65dfc0274da97c7308a677a36c6e`. Status: implementation and bounded acceptance complete; independent final
review pending. #721 access-log URL disclosure is
a separate later atomic ticket. #750 formatter delivery/cleanup is independent.

## Sequence

1. Reproduce the original #718 camelCase/bare-key client capture leaks and retain
   ordinary visibility, existing secret masking, passive body ownership and wire
   preservation controls. Reproduce #721 separately without changing its owner.
2. Trace evaluated security credential locations and mapped names through the
   shared HTTP transport IR and generated HTTP/JSON-RPC client/server wiring.
   Identify gaps for body, Basic roles, cookies, custom aliases and disabled auth.
3. Propose one diagnostic-redaction owner, including conventional-name fallback,
   explicit credential metadata and safe structured-body failure behavior. Client
   and server Debug should consume that owner; #721 later consumes its URL policy.
   Do not repair camelCase alone while authored custom credentials still leak.
4. Resolve actual policy uncertainty with root using concrete accepted/rejected
   examples. Distinguish development Debug visibility from default access logs.
   Consider service-wide conservative metadata versus per-request propagation,
   mutation/alias ownership, malformed query/body and incomplete capture paths.
5. Add focused direct/generator readiness witnesses and a bounded model when
   propagation/order/ownership interactions warrant it. Establish normal, edge,
   rejected and wire-unchanged acceptance before independent readiness review.
6. Only after approval, implement at the shared owner, update owning public and
   contributor guidance, compare bounded generated specimens byte-for-byte across
   parent/current/repeat, compile/wire-check relevant consumers, then freeze one
   exact final diff for independent review. Root owns integration/commit/closure.

## Initial source facts

- `http/client.go` owns a private string-name heuristic and recursive JSON/form
  redaction. It normalizes separators after lowercasing, missing camelCase and
  bare `key`. The generated example `NewDebugDoer` receives no authored metadata.
- `http/codegen/internal/transportir.Security.Parameters` already contains resolved
  credential `Name`, `In` and `SchemeName` from effective endpoint requirements.
  Body/Basic-role completeness and JSON-RPC adaptation still need verification.
- Server `http/middleware.Debug` currently prints raw URL, headers, route vars and
  captured body; it shares capture ownership but not the client's redaction owner.
- `clue/log.HTTP` and `Client` both print `req.URL.String()`; client error logging
  also prints raw errors and optional response bodies. #721 must avoid masking a
  URL field while disclosing the same credential through an error string.
- Public client Debug documentation promises conventional token/secret/API-key
  masking, omits malformed structured previews, and deliberately leaves ordinary
  non-structured bodies visible. Do not silently broaden that public body policy.

## Policy alternatives to evaluate

For #718, keep documented ordinary Debug visibility while adding deterministic
name fallback and explicit metadata for arbitrary credential aliases; metadata
must be immutable and must not change application requests or responses. A
service-wide aggregate can conservatively mask the same wire name across methods;
per-request metadata can be more precise but must reach diagnostics before capture
and remain correct across later response completion and caller context changes.

For later #721, default path-only access logs avoid unmodeled query secrets;
explicit ordinary-query allowlisting restores chosen diagnostics. The alternative
is preserving all query values except declared/heuristically sensitive names,
which cannot guarantee safety for arbitrary handwritten credential URLs. Record
consumer impact and ask root about the genuinely public default before changing it.

## Selected owner proposal

Root accepted the direction of an immutable aggregate provided it consumes the
existing semantic credential binding owner. Keep #721's default query policy
separate. Add a detached diagnostic projection at the expr binding boundary;
it must retain all credential roles and their source identities before emitting
physical names. Prepared request bodies then resolve JSON and form names through the existing
emitted representation tag owner, including empty-tag Go-field fallback.
Do not independently infer auth from field-name heuristics in a generator.

One public runtime policy owns conventional-name normalization and explicit
header/query/JSON/form field sets plus whole-body/path omission. Constructor inputs
are copied; policy state has no mutation API. Merge policies conservatively for a
service or selected CLI services. Known credential names are masked recursively in
structured objects/arrays. Scalar credential bodies are omitted; credential path
segments require whole-path omission unless a complete route-aware representation
is supplied. Missing body parsing support for a known credential body must omit
that preview, never print it as opaque data. Whole-path omission must clear both
Path and RawPath, and server route-variable diagnostics must apply the same
credential policy so they cannot disclose a hidden URL credential independently. Cookie headers are already wholly
sensitive. Ordinary opaque bodies remain visible when no credential-body metadata
applies. Neither request/response objects nor wire bytes may be changed.

Expose the generated transport policy for application wiring; generated example
clients pass their selected services' policies to Debug automatically. Both client
and server Debug consume the same runtime policy and preserve existing capture,
completion and bound owners. Service aggregation intentionally masks an ordinary
field if another method declares that wire name sensitive. This is preferable to
request-context publication state and keeps delayed EOF/Close logging independent
of later requests. No automatic server middleware activation is proposed.

## Readiness evidence and final acceptance

- Original conventional-name witness: 30 client cases fail; query/header/JSON/form
  and response JSON include camelCase and bare key. Seven actual server middleware
  cases fail, including Authorization and Cookie. Both test families assert
  unchanged application values and ordinary diagnostic visibility.
- A real generated client encoder module compiles and vets, then fails all four
  custom query/header/body and duplicated Basic-body redaction cases (`ticket`,
  `X-Permit`, `permit`, and `signin`/`proof`). This
  proves the issue survives actual encoding. Its module pin is verified pushed
  parent `v1.10.0-alpha.9.0.20261013051947-ee4552dc94ee`; local replacement exercises
  the candidate. Original red log is `/tmp/loom-718-generated-red.log`; the extended pushed-pin
  witness is `/tmp/loom-718-generated-basic-red.log` (2.863s).
- Direct prepared-IR control passes and demonstrates why using security parameter
  names alone is wrong: body logical `credential` is serialized as `permit`; Basic parameters report
  only Authorization even though explicit body fields emit both credential roles
  as `signin`/`proof`.
  Existing passive response completion and conventional redaction/capture tests
  pass under race (HTTP 1.450s, middleware 1.274s).
- TLA+ NamesOnly and LogicalName each fail at 17 states, Borrowed fails at26,
  Snapshot passes30. This abstracts classification and delayed completion; parser,
  I/O ownership, generator metadata and ordinary visibility require runtime tests.

Before final implementation review, extend direct metadata controls to inherited
security, NoSecurity, session cookies, explicit Basic-role body duplication, JSON
member remapping, scalar Body, path credentials, form and JSON-RPC. Exercise nested
JSON objects/arrays and conservative same-name masking, caller input mutation and
delayed completion, malformed/incomplete/oversized structured previews, unknown
content types carrying known body credentials, ordinary opaque bodies, repeated
query/header values and complete wire preservation. Match generated policy to
actual HTTP and JSON-RPC debug request/response paths. Preserve existing passive
capture/EOF/Close/error tests. Compare bounded parent/current/process-repeat
specimens covering custom mapped credentials and ordinary/disabled controls;
compile and vet each distinct generated output and list every intended difference.
No full fixture corpus and no production changes before readiness review.

## Production release

Root independently approved readiness manifest
`f37a1c969b4ad834c84db46ef3e14be30227cebaf17daf2da3162ac06ea0a01c`.
Production base fast-forwarded to pushed `0d9671589`; the direct metadata control
still passes and all four generated alias/Basic witnesses remain red after
build/vet (2.999s, `/tmp/loom-718-current-base-red.log`).

Implementation sequence: shared immutable runtime policy and both Debug consumers;
detached expr credential-binding projection and actual emitted JSON/form-name
owner; generated transport policy and automatic example wiring; then the promised
metadata/runtime/generated acceptance and exact independent final review. Public
configuration remains small: explicit credential field sets, copied at diagnostic
construction. No per-request mutation or alternate auth classification.

The detached binding projection also includes explicit query/header/cookie/path
operations whose source identities supply a credential operation. A Basic pair
duplicated into `signin` query and `X-Proof` header was otherwise missed; the
retained direct regression fails the initial projection. Authentication and
encoding continue to consume the existing unchanged binding plan.

## Final acceptance and recovery receipt

- [x] Shared immutable `observability/diagnostics.Policy` replaces the client-only
  redaction path and is consumed by both Debug implementations. The passive I/O,
  completion and 64 KiB retention owners are unchanged.
- [x] Effective credential source identities come from the existing expr wire
  binding plan, including duplicate Basic slots. Physical body names come from
  emitted representation tags. No new auth classifier or mutable request policy.
- [x] HTTP and JSON-RPC emit one service metadata artifact and example clients
  consume it. Handwritten client/server wiring is documented. Empty ordinary and
  NoSecurity services emit none. Service aggregation is intentionally conservative.
- [x] Conventional names, repeated headers/query/form, nested JSON, delayed EOF,
  caller-input mutation, request Content-Type mutation, malformed/incomplete and
  oversized captures, scalar bodies, opaque body policy, raw/opaque path and route
  values, and original wire bytes have direct/runtime controls. Full existing I/O
  and completion/error suites remain green. `NoSecurity` controls use ordinary
  fields, since security-role attributes themselves require an effective scheme.
- [x] Metadata tests cover inherited API/session security, cookie-only streaming
  and bidirectional sessions, mapped Basic username/password, duplicated query/
  header copies, body/form aliases, scalar Body and whole-path omission. Path API
  keys remain rejected by existing admission; the valid path witness uses an
  explicit Basic username copy. Form-only authored tags exercise JSON's actual Go
  field fallback; explicitly empty JSON tag names remain rejected by admission.
- [x] Generated HTTP query/header/body/Basic wire regressions pass; the final Basic
  control asserts both signin and proof wire values. Generated JSON-RPC envelope,
  form, custom physical JSON/form name and scalar preview tests pass under race.
  Both examples and all controls build and vet. Final focused runs:
  `/tmp/loom-718-freeze-wire.log` HTTP 4.934s, generator 14.162s.
- [x] Full affected owner races passed: diagnostics 1.194s, HTTP 11.891s,
  middleware 1.396s, expr 6.566s, representation 1.474s, transport IR 1.492s
  (`/tmp/loom-718-owner-gates.log`). Later opaque-path/source-name/refactor checks
  passed diagnostics 1.480s, HTTP codegen 5.693s, generator 13.597s
  (`/tmp/loom-718-final-changed-gates.log`). Relevant existing CLI/render/security
  controls passed 7.524s (`/tmp/loom-718-codegen-gates.log`). Scoped lint, including
  explicit codegen/testdata packages, reports zero issues (`/tmp/loom-718-lint.log`).
- [x] Three-design process-isolated comparison is recorded in
  `/tmp/loom-718-comparison/report.json`: 104 parent artifacts, 108 candidate and
  108 repeat artifacts, all repeated candidate bytes identical. All six distinct
  modules build/vet; each uses the verified pushed module pin with explicit local
  source replacement. Final post-refactor generation matches all 108 candidate
  artifacts (`final-report.json`, `/tmp/loom-718-final-comparison.log`). Subsequent
  signature wrapping and harness declaration ordering are mechanical only.
  Exactly seven differences are intended: four new service diagnostics files and
  three example CLI files adding metadata imports/arguments. Service types,
  encoders/decoders, server behavior and every OpenAPI artifact remain unchanged;
  the entire ordinary/disabled design is byte-identical. Body security methods are
  explicitly excluded from OpenAPI because that public format rejects body API
  keys; valid supported-location schemas remain in the comparison.
- [x] Model semantics are unchanged from reviewed readiness: NamesOnly and
  LogicalName fail at 17 states, Borrowed at26, Snapshot passes30. Sources moved
  beside the shared runtime owner at `observability/diagnostics/tla/redaction`;
  README and the formal index describe implementation correspondence and limits.
- [ ] Independent exact final review, root integration, atomic commit and closure.
- [ ] After delivery, retain compact manifests/receipts and remove inactive task
  build/model/raw-log artifacts; use managed worktree archive only.

### Separate residual, not bundled

Basic credentials additionally mapped into query/header slots produce four
pre-existing generated CLI redeclarations. The minimal evaluated design and
parent-generated output are retained under `/tmp/loom-718-basic-cli-parent`;
`/tmp/loom-718-basic-cli-parent-minimal-red.log` proves the failure on parent
`0d9671589`. The direct diagnostic projection for these duplicate slots passes;
our buildable generated matrix uses explicit Basic body copies. The independently reviewed report is filed as #842; its draft is retained at
`/tmp/loom-718-basic-cli-residual-issue.md`; #794 is a different default-pointer
failure. No CLI source-identity repair or diagnostic scope relaxation is included.


### Independent-review corrections

The original exact review found a medium disclosure: a handler can commit JSON
then change Content-Type to plain text before Debug reads its final headers.
The first proposed post-forward snapshot also failed when an inner writer changed
headers after committing the body. The retained regressions reproduce both failures
(`/tmp/loom-718-response-metadata-red.log` and
`/tmp/loom-718-forwarded-metadata-red.log`).

The shared `transport.CaptureResponse` now owns optional immutable metadata.
It snapshots before an observed final WriteHeader/Write/WriteString/Flush, retains
that snapshot only if forwarding leaves it unchanged, and marks opaque initial
ReadFrom metadata unavailable. Prior observed commits remain usable for ReadFrom.
Debug reads only this owner and omits an ambiguous preview. No early commit, extra
I/O, wrapper interface or wire change is introduced. Implicit content-type sniffing
is conservatively ambiguous; the bounded-capture visibility control now supplies
an explicit stable content type, and a dedicated direct control proves the sniff
case is omitted. Headers can be read concurrently through detached snapshots;
the underlying response writer retains its own serialization requirements.

`ResponseMetadata.tla` covers late mutation, opaque copy and forwarding mutation.
LateHeader/EarlyRead/AfterCall fail NoDisclosure; MetadataChecked passes 26 generated,
20 distinct states. The model explicitly excludes hidden mutate-and-restore inside
an arbitrary writer, and forwarding actions require the observable change checked
by the implementation. Tests cover original wire bytes, ordinary stable visibility,
header/write/flush, informational/upgrade and empty/committed reader paths.

Final correction gates:

- Full affected races: transport 1.436s, middleware 1.457s, HTTP 6.804s,
  diagnostics 1.144s (`/tmp/loom-718-finish-race.log`).
- Generated credential regressions under race: HTTP 5.022s and generator 12.296s
  (`/tmp/loom-718-finish-generated.log`). The later private ReadFrom hook extraction
  only lowers function complexity; the full runtime races cover that exact source.
- Scoped transport/middleware/testdata lint: zero issues
  (`/tmp/loom-718-finish-lint.log`). `git diff --check` passes.
- Earlier 104/108/108 generated-byte comparison remains valid: no generator
  production changes were made during correction. Three empty DSL callbacks were
  expanded mechanically, resolving the LOW finding; the stale parent above now
  matches the actual production base.
- Re-review pending on the frozen correction. No changes after authorization.
