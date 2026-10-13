# HTTP constructor ownership — #735

Status: implementation complete; bounded acceptance passed; final diff frozen for independent review. Readiness parent `521f7abc93d073b7d0283928e8dfbfd4986efbf3`; final parent `fae477d0e` (includes #853).

## Sequence

1. [x] Read issue #735, its shared-alias witness and current ownership guidance.
2. [x] Reproduce handler/payload, response/payload and shared target/different projected source collisions on this parent; retain ordinary controls.
3. [x] Trace package allocation, declaration deduplication and call identity across HTTP and JSON-RPC, including client-side constructor ownership.
4. [x] Model current behavior and structural alternatives: lexical uniqueness alone, semantic identity alone, shared package owner with complete conversion identity.
5. [x] Freeze source/evidence, proposed shared owner, affected consumer inventory and bounded acceptance for independent readiness review.
6. [x] After approval only, implement the owner and remove superseded allocation paths.
7. [x] Run focused regressions, bounded parent/candidate/repeat output comparison, generated compile/runtime and lint.
8. [x] Freeze the exact atomic final diff for independent review.
9. [ ] Independent final approval and root-owned atomic integration.

Scope: constructor declaration names and semantic conversion identity shared by
HTTP and JSON-RPC generated packages. #736, #770 and #786 remain separate unless
current evidence proves the same owner. No public policy question is apparent:
valid designs must compile, and a constructor may be reused only for the same
source/target conversion contract. No commits or changes to the frozen #740 tree.

## Current-parent findings

At `521f7abc9`, Go 1.27.2/jsonv2 with explicit local `LOOM_DIR`:

- Payload `Handler`/method `m`: `NewMHandler` conflicts with the handler factory
  in both HTTP and JSON-RPC.
- Payload/result `MResponseBody`: two incompatible `NewMResponseBody`
  constructors are emitted, in both transports.
- Two projected source types sharing selected `Text` response body: only the
  first `NewText` definition is emitted; the other method calls it with an
  incompatible view type. Both declaration orders fail in both transports.
- Two methods selecting `left`/`right` from the same PairView share parameter
  and return types, but require different conversion bodies. Both compile;
  generated encoders return `"left"` for the right method. Left is a passing
  runtime control. This is the same response-constructor identity defect.
- Distinct aliases, ordinary Input payload, and identical source conversion
  reuse compile on both transports. Existing client body naming/source tests
  provide further ordinary/edge controls to retain.

Inputs are `http/codegen/testdata/constructor_ownership.go`, direct/generated
HTTP readiness tests, and JSON-RPC generated tests. Logs:
`/tmp/loom-735-parent-final.log`, `/tmp/loom-735-parent-selector.log`.
Exact generated specimens are under `/tmp/loom-735-parent-artifacts`.
The historical #854 shared-alias source was copied independently to
`/tmp/loom-735-historical-shared-alias-dsl.go`; its receipt means no #854 temporary
tree is still needed. At this readiness checkpoint, no production files had changed.

## Owning seams and structural proposal

`http/codegen.ServicesData.analyze` prepares transport IR, allocates body type
names, analyzes endpoints and collects body/union types. JSON-RPC uses this same
analysis with its own HTTP expression set, then delegates type files to HTTP.
Constructor ownership belongs here, after all physical carrier/transform inputs
are available and before renderer/CLI consumers receive the analyzed service.
It must also run for the reanalysis performed by `FileData` import binding.

Current producers and consumers:

| Producer | Declaration/call consumers | Current defect |
| --- | --- | --- |
| `payloadBuilder.buildInitData`, WebSocket payload initializer | server type init; HTTP/RPC request decode and WebSocket receive | Formats name independently of handler/body constructors |
| `buildResponseBodyInit` | server type init; HTTP response/error, JSON-RPC response/error/SSE/WebSocket | Target-name-only identity ignores source signature and selected-field behavior |
| `buildRequestBodyInit`, WebSocket request body | client type init; request encoding and stream send | `clientBodyInits` owns only body constructors, separate from client result constructors |
| `buildResponseResultInit` | client type init; response/error decoders | Separate formatted name/dedup domain |
| endpoint `HandlerInit` plus other non-conversion declarations | HTTP/RPC server handlers and mount/dispatch wiring | Must be reserved before conversion names are allocated |

Proposed implementation sequence after readiness approval:

1. Introduce an internal finalized constructor plan in `http/codegen` containing
   the exact ordered parameters, return carrier and emitted conversion body.
   Build it using the same lower-level body/field/return emission used by
   `bodyInitSection` and `typeInitSection`; declarations and identity consume one
   plan. Do not maintain a second subset of InitData flags as an equality rule.
   Name, documentation, call-site argument expressions and examples are not part
   of a declaration's behavior. Conservatively distinct code plans may remain
   separate; semantic equivalence need not be inferred.
2. Add one per-service/per-output-package constructor registry, shared by all
   conversion roles. Reserve the actual non-conversion package declarations
   first, including fixed factories, endpoint handler factories, body/nested/
   union declarations, union factories/constants, validation functions and
   transform helpers. Keep server and client namespaces separate. Preserve
   existing preferred names when unoccupied; use NameScope for suffix allocation.
   Never use the neutral service scope or a new suffix rule in each producer.
3. After endpoint/union collection, finalize every constructor occurrence with
   the package registry. Identical finalized plans reuse one recorded name;
   different plans receive different names. Keep handler identities reserved
   rather than renaming handlers as a payload-specific workaround. Update each
   InitData occurrence so existing HTTP/RPC/stream/CLI calls read the allocation.
   Audit analysis-time embedded strings and retained copies before choosing the
   exact finalization seam; no call may retain a preferred-name string.
4. Separate constructor collection from body declaration deduplication.
   `serverTypeSections.appendTypeData` currently returns when a type was emitted,
   suppressing a later distinct constructor. Collect each planned constructor
   even for a reused type; emit each allocated definition once. Use the same
   constructor owner for client body/result init collection. Remove isolated
   `clientBodyInitKey`/`clientBodyInits` allocation and superseded constructor
   dedup rules; keep physical type/layout registries unchanged.
5. Inventory the renderer's package declarations against reservations with
   structural tests, especially occupied numeric suffixes and union factories.
   Public constructor names may change only where ownership requires it;
   document regeneration and direct-call migration. Schema/type renaming is not
   part of this ticket.

A signature-only identity is rejected: the left/right runtime failure has equal
argument and return types. A lexical-only fix is rejected: the renderer would
still discard a needed constructor with an already-emitted type. Merely removing
deduplication is rejected: existing same-plan reuse would redeclare functions.
The model checks all four counterexamples and the combined owner (4560 generated,
1630 distinct states). Concrete plan extraction and reservation completeness
remain implementation/test obligations, explicitly outside the model.

## Bounded acceptance and independent review gate

- Direct: original lexical collisions; differing projected parameter types;
  identical signature/different selected field; same-plan reuse; occupied suffix;
  server/client namespace separation; each potentially colliding `New*` non-conversion declaration
  represented in reservation inventory; non-`New` declarations are disjoint.
- Generated: both HTTP and JSON-RPC original three failures, shared-alias reverse
  order, same-plan and ordinary controls. Original selector runtime must encode
  the requested field, not simply compile. Add nil/empty/value checks for the
  shared optional alias and mounted HTTP/RPC response paths.
- Consumers: selected error response plus one WebSocket/SSE constructor path;
  existing clientBodyInitNames and PerSource controls (including flat/nested
  collections and WebSocket) compile and retain reuse. No service/interceptor or
  protobuf symbol repair (#770/#786/#736 have separate owners and scopes).
- Artifacts: retain bounded parent specimens above and one existing nontrivial
  stream/error/client-body fixture for parent/candidate/repeat comparison. Compile
  changed modules; list every intended constructor/call difference. Body/service
  declarations and OpenAPI contracts should be unchanged. If schemas change,
  investigate before expanding scope. Do not run the whole exported-design corpus.
- Finish affected package/direct tests, intentional goldens, scoped lint and docs;
  preserve source/evidence hashes and get fresh final review before one root-owned
  atomic commit. #740 remains untouched.

Readiness completion requires independent approval of this owner and its complete
identity before production changes. No public policy decision is pending.

## Implementation checkpoint

Readiness approved in `/tmp/loom-735-independent-readiness-verdict.json`. Advanced
tracked parent to `7eed12497`, preserving all eleven readiness inputs. Shared
constructor plans, package allocation and independent emission now pass the
original compile and selected-field runtime regressions. CLI declaration analysis
runs before allocation so its types, validators, union factories and helpers also
participate in the package inventory. CLI inputs retain references; no call text
is patched. ResponseEncodePlan is rebuilt from finalized InitData to compose
with #853 BodyValues. The final candidate composes onto delivered #853 (`fae477d0e`).

The bounded RPC AST inventory includes ordinary, mixed SSE/WebSocket viewed
streams and pure SSE. It asserts every conversion name starts `New` and that
every actual `New*` competitor is reserved. Lowercase RPC wrappers/decoders and
fixed StreamError declarations cannot intersect that constructor namespace; they
are explicitly excluded rather than introducing unrelated naming infrastructure.
HTTP ordinary/stream/multipart/relocated-union declarations retain the same
collision check. Root confirmed this bounded namespace rule during implementation.

## Final acceptance evidence

The bounded local-source command ran all `TestConstructor*` and
`TestClientBodyInitNames*` checks in HTTP and JSON-RPC codegen (9.250s / 5.653s).
It compiled sixteen original case modules and the existing relocated-types,
union, error and stream consumer design. Mounted shared-alias checks exercise
both declaration orders, required values and optional nil/empty/value results;
selector checks assert the actual selected field on the wire. Direct tests cover
complete plan identity, occupied suffixes, namespace separation, same-plan reuse,
actual HTTP/RPC declarations and import-alias reanalysis. The #853 response-plan
check requires every captured call to use the final allocated constructor.

The exact parent generator was rerun with a Go source overlay over this same Git
worktree and identical generation inputs. Changed production files were replaced
with parent blobs; new production files became empty package stubs and tests
requiring the new owner were excluded through empty test stubs. The generator
change has no runtime edits, so generated module builds use the same runtime
source in parent and candidate. Parent reproduces eight compilation failures
and two wrong-field response failures. Ordinary, distinct-alias, same-plan and
consumer controls pass. This boundary avoids treating a copied non-Git source
directory as a valid temp-module harness root.

All three generated artifact sets contain 264 files. An independent process
repeated generation byte-for-byte. Exactly 28 Go files differ from the parent:

- handler collisions allocate `NewMHandler2` and update request decode calls;
- response/payload collisions allocate `NewMResponseBody2` and update response
  calls, including JSON-RPC's captured framing expression;
- differing projected inputs and equal-signature/different selectors emit
  `NewText2` with the correct parameter/body and update its calls;
- identical client result plans reuse the first allocated constructor, removing
  the redundant declaration and updating the second response decoder.

All 18 JSON/YAML artifacts are unchanged, as are the 46 files from the existing
relocated-types/union/error/stream consumer. No schema validation expansion was
needed. The only checked-in golden change likewise reuses an identical empty
client result constructor. Both checked-in ticktock SSE fixtures regenerate
byte-for-byte and build (HTTP 9.661s; JSON-RPC 9.081s). Existing generated
WebSocket named-payload and SSE viewed-result runtime tests and all ClientDecode
goldens pass (5.999s). Scoped HTTP/RPC/testdata lint reports zero issues.

Evidence lives in `/tmp/loom-735-final-manifest.json` and its hashed logs,
parent-overlay map, generated comparison and exact patch. Model source and
assumptions are unchanged from independently approved readiness: the four
negative designs fail their stated invariant and the combined owner passes
4560 generated / 1630 distinct states. These results are reused, not rerun merely
because unrelated source advanced.

An exploratory full-package run is **incomplete evidence**: HTTP reached the
suite's ten-minute timeout, while JSON-RPC exposed previously existing SSE
`Recv(ctx)` harness and stream-golden mismatches. Root owns these separate #720
CI repairs; they are not silently repaired or counted as passing #735 acceptance.
The bounded checks above establish this ticket's changed behavior.

## Final review follow-up

Independent final review found no medium/high correctness issue and verified all
792 parent/candidate/repeat artifact hashes. Two low style findings were repaired
mechanically: remove body/type constructor forwarding wrappers and call the
shared renderer directly; expand new nonempty single-line test and raw-harness
blocks. The six call sites retain their section names, client/body flags and
package scope arguments. No assertions or conversion semantics changed.

The narrow follow-up reruns constructor/client-naming tests, the two modified
union-response seam calls, mounted harnesses and scoped lint. Generated output is
compared with the already reviewed candidate; model, parent and independent
repeat evidence remain valid. The exact style delta and follow-up hashes are
recorded in the updated final manifest for the original reviewer's confirmation.
