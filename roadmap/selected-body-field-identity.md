# Selected body field identity — #847

Parent: pushed `99e2c33d6`. Readiness passed independent review; implementation
and bounded acceptance are in progress. The accepted DSL
`struct:field:name` contract determines the service field declaration; selecting
that field as a body must retain the same physical identity through constructors,
selectors and CLI input conversion.

## Sequence and acceptance

1. Read the framework skill, inspect existing selected-body and physical-field
   owners, and reproduce the filed scalar request witness before implementation.
   Inventory every `Goify(origin)` and `serviceBodyTransformSource` consumer in
   http/codegen, including ordinary/selected response and error paths.
2. Extend a shared meaningful design fixture for request primitive, named,
   optional, nullable, object and collection fields, and selected result/error
   controls. Exercise both HTTP and JSON-RPC only where that transport supports
   the selected-body contract. Retain ordinary names and whole-body controls.
3. Assert logical field selection and physical declared field independently in
   analysis data. Compile representative generated modules to distinguish a real
   identity defect from merely different printed output. Record all failures and
   controls; do not weaken existing selected-body behavior.
4. Propose one shared selected-field binding that carries logical origin and
   physical source member together. Reuse the existing metadata-aware field-name
   owner; downstream request/response/body/CLI renderers consume the resolved
   binding rather than converting a logical name again. Inventory reference use
   before choosing the record shape. Public tags and wire values stay unchanged.
5. Model the logical/physical identity distinction if the chosen cross-consumer
   representation needs new state. Freeze readiness with exact source hashes,
   rejected alternatives, intended output changes and bounded implementation
   obligations; require independent readiness review before production edits.
6. After approval, add runtime round trips and missing/invalid controls, then
   implement in the shared owner. Compare parent/candidate/fresh-process generated
   artifacts byte for byte, compile and vet selected modules, run affected direct
   suites, and independently review the exact final diff before one atomic
   commit, push and issue closure.

## Current ownership evidence

`transportFieldBinding` already resolves ordinary wire fields with
`GoifyAtt(svcField, name, true)`. Selected-body data instead separately calls
`Goify(origin, true)` in payload, response and body constructors, while
`serviceBodyTransformSource` builds a selector from the same logical name.
Logical origin remains necessary for effective requiredness/default selection;
replacing it globally with a physical name would break those semantics. Keep
those identities separate and derive all physical uses from the declared member.

## Progress

- [x] Read owning source and original failing non-auth request evidence.
- [x] Reproduce and classify the bounded request/response/error matrix.
- [x] Freeze and independently review shared-owner readiness.
- [x] Implement, compare generated output and run runtime controls.
- [ ] Independent final review, atomic commit, push and closure.

## Confirmed readiness matrix

Both transports reproduce renamed selected request and response failures with
unknown `Value` members while the declaration contains `ActualValue`. Renamed
named/optional/defaulted/nullable/array/object request bodies also fail at the
same physical selector/constructor boundary. Seven direct HTTP cases report the
wrong PayloadAttr; the compiler witnesses independently check emitted uses.

The 28 generated modules yield exactly 16 expected compile failures and 12
passing controls: ordinary selected request/response, ordinary/renamed custom
errors, and ordinary/renamed whole bodies in each transport. The error fixture's
Type must be declared at top-level; the initial invalid nested-Type probe was
corrected before the accepted final log. Baseline evidence is
`/tmp/loom-847-readiness-red.log`; it predates the implementation described below.

Proposed #847 ownership boundary (readiness, no production changes):

Existing GoifyAtt owns physical member naming. A private selectedServiceField
record in http/codegen retains logical name, actual object key, raw declared
attribute, effective transform attribute and Go member name. Resolve it from the
service payload/result and selected logical Body origin, never the copied HTTP
body type. Keep empty selection as whole value without a field selector.

The selector-producing serviceBodyTransformSource and request/response analysis
use this record. RequestData.PayloadAttr and InitData.ReturnTypeAttribute derive
from its physical name. Requiredness, selected defaults, union value-vs-pointer
and optional-body tests retain the object key/logical origin. No change to JSON
name, transport body schema or runtime value contract is intended.

Exact inventory:
- buildRequestData: PayloadAttr currently Goify(origin).
- buildInitData: ReturnTypeAttribute currently Goify(origin) returned by buildTransformCode.
- buildSingleResponseData: ResultAttr currently Goify(origin).
- buildResponseResultInit: ReturnTypeAttribute currently Goify(origin).
- buildRequestBodyInit/buildResponseBodyInit: ReturnTypeAttribute currently Goify(origin)
  from serviceBodyTransformSource, whose selector also uses Goify(name).
- service_data_json_input.go already selects target with logical BodyOrigin; keep
  this lookup logical, then preserve physical return-field data from original init.
- bodyFieldIsUnionValue/buildResponseResultPointer/optionalBodyDefault continue
  semantic key lookup. responseTagData is a different selector policy, not in #847.
- selected custom errors currently compile through their existing dedicated
  builder and are retained controls, not claimed broken.

No new exported API. Reuse GoifyAtt and serviceFieldTransformAttribute; remove
repeated raw Goify projections. Shared HTTP analysis already serves JSON-RPC.

## Implementation acceptance after readiness review

- Direct analysis checks retain original logical BodyOrigin and required/default
  lookups while all physical PayloadAttr/ResultAttr/ReturnTypeAttribute fields
  match the service declaration. Include a mapped-key control and a selected
  viewed response to cover the representation boundary beyond ordinary fields.
- Every one of the 28 bounded modules must build/vet after repair. Compiled runtime
  probes must send and decode selected request/result values through both
  transports, test CLI selected body construction, default-on-absence, optional
  absence, nullable null/value and invalid-body rejection. Custom error and whole
  body controls preserve their current result.
- Generate these fixtures at parent/candidate/fresh-process candidate, with the
  parent tests altered only to supply identical fixture inputs. Intended changes
  are selected-body Go selectors and initializer keys that refer to renamed
  members; service declarations, OpenAPI wire contracts, unrenamed and whole-body
  output must remain byte-identical. Compare complete generated artifacts, not
  decoded approximations. No full exported-design audit is required.
- Run direct selected-body suites and existing nullable/optional request and
  selected response/error tests affected by the shared helper. Run relevant lint,
  model configs and exact final independent review. Record concrete commands
  and hashes before root integrates one atomic #847 commit.

The design comparison model reproduces logical-only PhysicalIdentity failure
(four generated states) and physical-only SemanticIdentity failure (three).
The resolved record passes all six bounded states. This confirms the identity
separation rationale only; valid-field admission, metadata interpretation,
viewed layouts and generated code remain explicit Go-test obligations.


## Implementation checkpoint and independent prerequisites

The private selectedServiceField record now owns logical Name, actual Key,
metadata-aware GoName and effective Attribute. Raw declarations are used only
while resolving GoName; do not retain a second unused raw attribute. Payload,
body and response analysis consume the same record. The old pass-through request
requirements and repeated raw Goify projections are removed. Shared GoifyAtt
remains the naming authority; no new public API or wire policy is introduced.

Original acceptance: all 28 generated modules build/vet. Mounted HTTP/JSON-RPC
request, HTTP response, selected CLI, malformed-input and HTTP absence/default
checks pass across primitive, named, optional, defaulted, nullable, array and
object carriers (`/tmp/loom-847-runtime-third.log`, HTTP35.024s/RPC31.198s).
Optional raw-string CLI empty text retains its existing absence convention;
mounted transport tests separately verify explicit empty strings. Malformed
requests assert status, problem category/detail and no service invocation.
Additional selected response codec controls pass for JSON-RPC, explicitly below
its faulty mounted framing layer. Direct mapped-key tests preserve logical
selection, requiredness, default and the custom physical member. Existing
selected/default/nullable/CLI/problem tests pass (HTTP69.050s).

Two new independent reports prevent claiming all originally planned cases pass:

- #853: even ordinary JSON-RPC selected scalar responses send the whole service
  result through the mounted success handler. Direct generated encoder/decoder
  controls pass; they do not establish mounted behavior. The original failing
  harness is `/tmp/loom-847-first-runtime-harness.go`; logs are
  `/tmp/loom-847-runtime-second.log`. The standalone remote-pinned report was
  independently reproduced, reviewed and filed.
- #854: required selected scalar ResultType responses generate a plain scalar
  assignment into projected pointer storage in both clients. The required
  ordinary/renamed compiler witnesses remain unchanged in the test matrix.
  Additional optional viewed controls compile, but do not replace required
  acceptance. Preserve `/tmp/loom-847-viewed-compile.log` and the retained failure
  log. The separate storage repair must land before this ticket's final review.

The first 32-design output comparison includes the failing required-viewed
specimens: 420 artifacts per revision, exactly 50 physical-name reference changes
and a byte-identical independent candidate repeat. Specs, service declarations,
ordinary fields and whole-body outputs remain identical. The later optional-view
comparison is supplementary; neither output comparison proves compilation of
required views. Hash maps preserve each capture. After #854 integration, rerun
only invalidated selected-view and response constructor checks and the affected
artifact comparison against the new parent, retaining prior valid evidence.

Touched code passes scoped HTTP/JSON-RPC lint. Explicit testdata-package lint
also exposes a pre-existing maintidx failure in BasicCLISourcesDSL from #842;
new selected-field fixture/harness warnings were corrected structurally. Do not
claim the complete lint tier green until that separate fixture failure is repaired.
Final exact-diff review and integration remain pending; no #847 closure claimed.

## Integration after prerequisite delivery

The candidate is now being completed on main after `d4b2905d8`. The original
worktree at `56c46446f` remains untouched as recovery evidence; its source copy
and transfer hashes are `/tmp/loom-847-transfer-source` and
`/tmp/loom-847-transfer-manifest.json`. Three-way integration retained #854's
canonical physical-storage query and guarded source conversion while changing
only the selected-member identity. #858 now emits required view packages from
shared test-module assembly, so the uncommitted caller compensation was removed.
#855's fixture lint repair is also delivered. Historical failure and artifact
captures above remain evidence of their original parents, not current acceptance.

Next run the original required-view matrix and full bounded carrier/runtime
matrix, regenerate parent/current/repeat artifacts at this new base, and freeze
for fresh exact-diff review. #853 mounted JSON-RPC projection remains separate.

## Final bounded acceptance

At pushed production base `d4b2905d8`, both ordinary required-view controls build
and vet; the renamed request and required-view witnesses still fail in both
transports with incorrect Value selectors/keys. The candidate's original
32-module matrix plus four optional-view controls passes, preserving mounted
request/CLI, malformed input, absence/default/null/value and HTTP response checks.
HTTP45.362s and JSON-RPC45.622s include generated race tests and vet. Shared #854
mixed constructor/mounted storage control passes5.453s. Scoped HTTP/JSON-RPC and
service fixture lint reports zero issues. No runtime assertion was removed.

Each parent/candidate/repeat generation contains420 artifacts. Exactly50 change:
physical references in renamed selected-body constructors, selectors and CLI
conversions. All specs, service declarations, ordinary-name controls, error and
whole-body output remain unchanged. Candidate equals fresh-process repeat byte
for byte. The complete hash map and enumerated changes are
`/tmp/loom-847-current-artifact-hashes.json`; exact changes are
`/tmp/loom-847-current-artifacts.patch`. Earlier model source/configurations are
unchanged; their independently reviewed three outcomes remain applicable.
The later #860 commit `58defc6c7` changes only unrelated handwritten name harnesses
and its plan, so these generator-source results remain valid on the final parent.

Final review scope is this selected-member identity repair and its guidance,
fixtures and model. Preserve #854's canonical pointer owner and #858's shared
view assembly. No new public policy is introduced: documented struct:field:name
metadata is honored when a field is selected as a body. Regenerate affected
clients/servers. #853 remains an explicitly failing mounted JSON-RPC projection
contract; direct encoder controls in this ticket do not claim to fix it.
