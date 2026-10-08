# Formal models and proof maintenance

Use this index when changing a modeled behavior. Follow the linked local README
for the exact checker command, toolchain, configuration, expected counterexample,
assumptions and implementation tests. Those files own run details and results;
this index does not duplicate changing state counts or theorem totals.

Models complement direct, rendered, generated-build and runtime tests. A bounded
TLC run checks only its configured state space. A Lean theorem establishes its
stated predicates under its hypotheses; it does not establish handwritten Go
refinement, correct plan extraction, arbitrary codec behavior or generated-code
correctness. Keep proved, tested, bounded-checked and assumed claims distinct.

## Value semantics and expression ownership

| Code concern | Model and correspondence |
| --- | --- |
| Source selection/resolution, branch identity, presence, canonical wire construction and target preservation | [Lean value contract model and proofs](../../../../expr/lean/value_projection/README.md), [correspondence ledger](../../../../expr/lean/value_projection/correspondence.md), and [generated comparison harness](../../../../internal/valuecontract/README.md). The README maps executable stages to independent specifications and proof modules; the ledger distinguishes checked candidate claims, pending obligations and tested production boundaries |
| Value source precedence, phase ordering, representation and cache ownership | [Value pipeline TLA model](../../../../expr/tla/value_projection/README.md); bounded ownership checks, not arbitrary value or generated-program correctness |
| Source/target pairing, declaration identity, baseline acquisition and annotation paths | [Schema ownership models](../../../../expr/tla/schema_declaration/README.md); bounded checks reject incompatible baseline reuse, added samples and erased annotations; actual extraction, recursion, naming and projection remain Go test obligations |
| Request-body analysis, method type ownership, inherited/default result views | [Expression lifecycle models](../../../../expr/tla/README.md); local README maps them to expression and generated transport tests |
| Copying incomplete union branch occurrences, mutable metadata ownership, recursive expansion | [Union copy models](../../../../dsl/tla/union_copy/README.md) |
| Promoted union identity, authored-name reservations and deterministic allocation | [Union naming model](../../../../dsl/tla/union_names/README.md) |
| Untagged JSON schema/decoder agreement, selected identity and transactional assignment | [Untagged matching model](../../../../pkg/tla/untagged_json/README.md) |
| JSON options at nested union/optional/nullable codec boundaries | [JSON options model](../../../../pkg/tla/json_options/README.md) |

## Generated declarations and transport mappings

| Code concern | Model |
| --- | --- |
| Service/view union references and declaration identity | [Service union declarations](../../../../codegen/service/tla/union_declarations/README.md) |
| View union allocation before conversion references | [View union names](../../../../codegen/service/tla/view_union_names/README.md) |
| Relocated type declarations, local reservations and union helpers | [External type names](../../../../codegen/tla/external_type_names/README.md) |
| Final client-streaming result interception and selected views | [Final result interception](../../../../codegen/service/tla/README.md) |
| HTTP response first-match priority and default placement | [Response order](../../../../http/codegen/internal/transportir/tla/response_order/README.md) |
| HTTP body wrapper copy/example identity and generated type names | [Body type names](../../../../http/codegen/tla/body_type_names/README.md) |
| HTTP type collection, validator eligibility and multipart cycle identity | [HTTP type identity](../../../../http/codegen/tla/type_identity/README.md) |
| HTTP union declarations matching allocated references | [HTTP union declarations](../../../../http/codegen/tla/union_declarations/README.md) |
| HTTP WebSocket payload null versus end-of-input framing | [WebSocket payload nullability](../../../../http/codegen/tla/README.md) |
| Protobuf endpoint/anonymous message name reservations | [Message names](../../../../grpc/codegen/tla/message_names/README.md) |
| Protobuf explicit names, collection wrappers, compatible declarations and recursive normalization | [Message declarations](../../../../grpc/codegen/tla/message_declarations/README.md) |
| Protobuf scalar presence, requiredness and defaults | [Scalar presence](../../../../grpc/codegen/tla/scalar_presence/README.md); bounded omission/zero checks, with real protobuf and generated decoder/stream tests |
| Protobuf fields, getters, synthetic oneofs and map-entry wrapper names | [Field names](../../../../grpc/codegen/tla/field_names/README.md) |
| Generated import aliases and metadata local-variable reservations | [Import aliases](../../../../grpc/codegen/tla/import_aliases/README.md) |
| Aggregate CLI import, usage and flag-set identifiers and their references | [CLI identifiers](../../../../codegen/cli/tla/identifiers/README.md); bounded allocation safety, with renderer reservations and per-server isolation checked in Go |
| Protobuf named-map oneof branches, including selected empty maps | [Map union branches](../../../../grpc/codegen/tla/map_union_branches/README.md) |
| JSON-RPC stream view selection and validation | [Stream/view decoding models](../../../../jsonrpc/codegen/tla/README.md) |

These local READMEs identify the generator seams, direct regressions and generated
compilation or round-trip checks that connect the abstractions to production.
Name-allocation assumptions are not proved merely by proving declaration lookup.

## OpenAPI contracts

| Code concern | Model |
| --- | --- |
| Nested synthesized union selection and occurrence-specific example caches | [Nested union examples](../../../../http/codegen/openapi/internal/ir/tla/nested_union_examples/README.md) |
| Alias-bound intersection and decoded Bytes length projection into base64 schema branches | `AliasLengthBounds.lean` and `ByteLengthProjection.lean` in the [Lean value model](../../../../expr/lean/value_projection/README.md); [production owners and checks](../../../../internal/valuecontract/BYTE_SCHEMA.md) distinguish proved arithmetic from tested codec extraction and schema allocation |
| One wire representation for byte/text branch comparison | [Byte example projection](../../../../http/codegen/openapi/internal/ir/tla/byte_examples/README.md) |
| Security binding ownership, explicit reservations and deterministic component names | [Security bindings](../../../../http/codegen/openapi/internal/ir/tla/security_bindings/README.md) |
| Complete recursive representation equivalence before component naming | [Representation equivalence](../../../../http/codegen/openapi/internal/ir/tla/representation_equivalence/README.md); bounded partition-refinement and quotient-fingerprint checks with legacy and overmerge counterexamples |

The historical byte-example model's encoding and enum assumptions do not prove
actual decoder uniqueness. Consult the current [value correspondence ledger](../../../../expr/lean/value_projection/correspondence.md)
for byte-length, alias, schema/runtime and representation-ownership weaknesses.
Rendered schema validation and actual decoder behavior are separate obligations.

## Runtime and process protocols

| Code concern | Model |
| --- | --- |
| Service-error merge and history snapshot ownership | [Error ownership](../../../../pkg/tla/error_ownership/README.md); bounded aliasing counterexamples, with repeated merges and protobuf reconstruction checked in Go |
| gRPC aggregate error status and detail ownership | [Error contracts](../../../../grpc/tla/error_contract/README.md); bounded branch-order and complete-failure checks, with wrappers and generated unary/streaming mappings tested in Go |
| Basic-auth header presence, component projection and required rejection | [Basic presence](../../../../http/codegen/tla/basic_auth/README.md); bounded required/optional and absent/empty/value combinations |
| Escaped route literals and split-before-unescape path arrays | [Path decoding](../../../../http/tla/path_decoding/README.md); bounded operation-order checks, with raw capture preservation and actual dispatch checked in Go |
| gRPC opening-send EOF and final status ownership | [Initial send model](../../../../grpc/codegen/tla/initial_send/README.md); bounded stream retention and receive-completion checks |
| JSON-RPC mount/Use order and configured handler dispatch | [Handler dispatch](../../../../jsonrpc/codegen/tla/handler_dispatch/README.md); configuration-before-requests contract with mount-order counterexample |
| JSON-RPC WebSocket response routing, client closure/redial and closure error identity | [JSON-RPC connection models](../../../../jsonrpc/tla/README.md) |
| Redis job ownership, fencing/resume, requeue replies, event acknowledgments and join reconciliation | [Pulse pool models](../../../../pulse/pool/tla/README.md) and their linked ownership design/action map |
| Stream group creation/removal, per-stream map ownership and consumer rotation | [Pulse streaming models](../../../../pulse/streaming/tla/README.md) |
| Integration child-process ownership, parent death and launch races | [Process lease model](../../../../internal/testprocess/tla/README.md) |

Runtime models state atomicity, fairness, Redis and operating-system assumptions.
Keep their real runtime/adversarial tests; an abstract process-group kill or Redis
transition does not prove a system call or server implementation.

## Keep proofs current

1. When a counterexample, review or runtime test exposes a weakness, identify the
   affected theorem, model assumption or production-correspondence claim. Mark
   that broader claim pending; do not erase valid narrower lemmas or report their
   old approval as evidence for the newly exposed case.
2. Add or refine a durable counterexample/model and the relevant ledger or local
   README. Reproduce the old/rejected behavior before claiming the candidate
   repair. Record the actual source seam and direct regression where available.
3. Adjust candidate definitions, hypotheses, required-theorem manifests and
   production tests as needed. A new codec assumption or an excluded value class
   must be explicit; it cannot silently make progress or preservation vacuous.
4. Re-run the local audited gate, expected-failure controls and affected
   implementation checks. Obtain independent exact-diff re-review before a
   dependent implementation relies on the updated result.

The Lean value gate audits transitive axioms and performs fresh kernel replay;
follow its local README rather than substituting a successful compilation. TLC
legacy configurations often must fail a named invariant: distinguish that
counterexample from missing tools, resource exhaustion or a malformed model.

## Retain sources; remove task output

Repository [AGENTS.md](../../../../AGENTS.md) owns this manual cleanup process.
It is a task-completion responsibility, not an automatic build-script action.

Keep durable model/proof source, checker configurations, required-theorem
manifests, toolchain/dependency pins, run instructions and concise findings with
their proved/tested/assumed limits. Keep actual regression test source in its
owning test suite. Record the source identity, commands, expected and observed
outcomes, counterexample meaning and unresolved obligations needed to reproduce
the result; a temporary log path alone is not durable evidence.

Do not commit compiled/checker output: Lean `.lake/`, `.olean`/`.ilean` files,
generated object files or executables; TLC state directories and generated trace
files; temporary probe modules, comparison snapshots or verbose scratch logs.
This does not remove intentional checked-in generated regression fixtures.

After a task's validation and independent review are complete, preserve the
concise findings above, then remove its owned compiled/checker/temp outputs.
Before removal, confirm no active process, reviewer or dependent task still
needs them. Defer cleanup of a shared prerequisite until its last active
consumer finishes. Use exact task-owned paths, never broad deletion patterns;
do not remove proof sources, another task's artifacts, shared toolchains or
shared Go caches. Existing repository restrictions on Git cleanup and cache
clearing remain in force. This index records the lifecycle rule, not permission
to delete an unknown directory.
