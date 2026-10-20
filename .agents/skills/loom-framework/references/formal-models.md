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

## Shared generated declarations

The [declaration merge model](../../../../codegen/generator/tla/declaration_merge/README.md)
checks declaration coverage, duplicate admission and effect occurrence retention
across contribution orders. Direct AST/admission tests and compiled shared-package
specimens establish the concrete Go identities and const/init behavior.

## CLI dependency ordering

The [CLI argument model](../../../../http/codegen/tla/cli_arguments/README.md)
checks selected service order and shared declaration/call dependency identities,
with grouped-argument and unfiltered-service counterexamples. Direct identity
and emitted-signature tests complement generated HTTP/JSON-RPC compilation.

## Value semantics and expression ownership

| Code concern | Model and correspondence |
| --- | --- |
| Native gRPC server handler, admission and embedding roles | [gRPC receiver members](../../../../grpc/codegen/tla/server_members/README.md); current, partial and declaration-only counterexamples, complete receiver ownership and generated registration/import controls |
| HTTP/JSON-RPC client factories, builders, Doers and mixed-stream roles | [Client member namespaces](../../../../codegen/tla/client_members/README.md); fixed-only counterexample, complete conditional allocation, generated compiler and mounted dispatch controls |
| HTTP/JSON-RPC server fields, copied WebSocket roles and canonical method independence | [Server member namespaces](../../../../codegen/tla/server_members/README.md); current and Server-only counterexamples, shared receiver allocation, generated compiler and mounted routing controls |
| Source selection/resolution, branch identity, presence, canonical wire construction and target preservation | [Lean value contract model and proofs](../../../../expr/lean/value_projection/README.md), [correspondence ledger](../../../../expr/lean/value_projection/correspondence.md), and [generated comparison harness](../../../../internal/valuecontract/README.md). The README maps executable stages to independent specifications and proof modules; the ledger distinguishes checked candidate claims, pending obligations and tested production boundaries |
| Exact numeric source preservation, interval comparison, native thresholds and schema annotation provenance | [Numeric Lean models and checked extraction](../../../../expr/lean/value_projection/README.md), with source grammar, cross-language token, generated transport and retained enum tests in the correspondence ledger. Proofs do not establish parser correctness or arbitrary external numeric decoding |
| Local and inherited numeric/length domain admission | [Constraint domain admission](../../../../expr/tla/constraint_domain/README.md); local-only and masked-negative counterexamples, exact endpoint and DSL checks; excludes machine-carrier satisfiability |
| Prepared body enum declaration provenance and mapped member identity | [Prepared enum provenance](../../../../expr/tla/prepared_enum_provenance/README.md); six bounded failure modes and retained-declaration invariants, paired with mapped/body construction, generated wire, schema identity and exact-output checks |
| Required annotation admission across text, raw and custom target codecs | [Schema annotation codec boundary](../../../../expr/tla/schema_annotation_codec/README.md); bounded false-rejection and blanket-fallback counterexamples, with direct sibling checks and rendered cookie/SSE contracts |
| Primitive numeric source admission and carrier normalization | `NumericValueAdmission.lean` and the source-resolution claims in the [Lean value contract](../../../../expr/lean/value_projection/README.md); independent carrier-role conformance checks exact integer limits and retained float/source rules. Redundant bounds remain predicates rather than stored carrier values |
| Value source precedence, phase ordering, representation and cache ownership | [Value pipeline TLA model](../../../../expr/tla/value_projection/README.md); bounded ownership checks, not arbitrary value or generated-program correctness |
| Intrinsic result nullability, occurrence ownership and selected-view declaration reuse | [Projected nullability](../../../../expr/tla/project_nullability/README.md); declaration-loss, occurrence-promotion, view-conflation and source-mutation counterexamples paired with semantic JSON and actual DSL tests; physical generated carriers remain a separate obligation |
| Source/target pairing, declaration identity, baseline acquisition and annotation paths | [Schema ownership models](../../../../expr/tla/schema_declaration/README.md); bounded checks reject incompatible baseline reuse, added samples and erased annotations; actual extraction, recursion, naming and projection remain Go test obligations |
| Request-body analysis, method type ownership, inherited/default result views | [Expression lifecycle models](../../../../expr/tla/README.md); local README maps them to expression and generated transport tests |
| Extend composition before validation, detached occurrence ownership and authored overlays | [Effective design composition](../../../../expr/tla/effective_design/README.md); bounded overwrite/order/aliasing counterexamples paired with direct and generated refinement tests |
| Resolved union branch names and discriminator identities before generator capture | [Union branch admission](../../../../expr/tla/union_branch_admission/README.md); current, early, single-identity and global-scope counterexamples, with evaluated and CLI rejection tests |
| Copying incomplete union branch occurrences, mutable metadata ownership, recursive expansion | [Union copy models](../../../../dsl/tla/union_copy/README.md) |
| Promoted union identity, authored-name reservations and deterministic allocation | [Union naming model](../../../../dsl/tla/union_names/README.md) |
| Untagged JSON schema/decoder agreement, selected identity and transactional assignment | [Untagged matching model](../../../../pkg/tla/untagged_json/README.md) |
| JSON options at nested union/optional/nullable codec boundaries | [JSON options model](../../../../pkg/tla/json_options/README.md) |

## Generated declarations and transport mappings

| Code concern | Model |
| --- | --- |
| Nearest authentication scope, session lowering and contradictory declarations | [Authentication scope ownership](../../../../expr/tla/authentication_scope/README.md) |
| Effective HTTP request/response/error bindings, credential framing and inherited mapping ownership | [HTTP wire bindings](../../../../expr/tla/http_wire_bindings/README.md); bounded collisions and valid disjoint auth alternatives |
| Recursive view selection, excluded-field preservation and semantic conversion helper identity | [Recursive selected-view ownership](../../../../codegen/service/tla/recursive_projection/README.md); bounded selection/order models paired with generated unary and streaming wire regressions |
| HTTP streaming body selection and omitted mapped SSE data | [Selected-view stream bodies](../../../../http/codegen/tla/sse_view_body/README.md); first-body and absent-field counterexamples with generated SSE/WebSocket controls; assumes sequential view selection |
| Service package declarations, error factories, method members and method-derived helpers | [Package symbol ownership](../../../../codegen/service/tla/package_symbols/README.md); missing reservations, shared-method-scope, unowned-helper and uncached error-factory counterexamples, with generated HTTP/gRPC/JSON-RPC controls |
| Authored error names before method/service inheritance | [Error declaration admission](../../../../expr/tla/error_declaration_admission/README.md); current, post-merge and global-name counterexamples with scoped DSL/CLI rejection and cross-scope override controls |
| Service/view union references and declaration identity | [Service union declarations](../../../../codegen/service/tla/union_declarations/README.md) |
| Complete recursive validator definitions, occurrence calls and diagnostic paths | [Recursive validation](../../../../codegen/tla/recursive_validation/README.md); bounded partial-buffer/path counterexamples, with generated finite-tree and named-nullable carrier controls |
| Collection references to allocated service and external declarations | [Collection identity](../../../../codegen/tla/collection_identity/README.md); raw-name and blanket-local-allocation counterexamples, with generated collision/reservation round trips and external-package controls |
| JSON automatic omission and physical carrier/projection agreement | [Field omission](../../../../codegen/tla/json_omission/README.md); current, global-zero and tags-only counterexamples; native/custom carrier, generated HTTP/RPC and rendered OpenAPI controls |
| Named validator physical identity versus occurrence-only presence | [Validation identity](../../../../codegen/tla/validation_identity/README.md); dropped metadata and naive presence-copy counterexamples, with shared/protobuf scope and generated three-transport checks |
| Unwrapped anonymous/named object carriers versus child field layout | [Validation carrier agreement](../../../../codegen/tla/validation_carrier/README.md); bounded invalid-guard and cleared-layout counterexamples, with generated HTTP/JSON-RPC enum and nested-object controls |
| Named validator eligibility across physical field layouts and structural union presence | [Validation layout](../../../../codegen/tla/validation_layout/README.md); caller-layout closure and omitted structural-presence counterexamples, with nested-union runtime and both-order shared-attribute controls |
| Selected sum-union branch storage and validator invocation | [Union branch carrier](../../../../codegen/tla/union_branch_carrier/README.md); nested-pointer panic, nil-collection false rejection and guard-only bypass counterexamples, paired with direct context and generated runtime controls |
| Selected union branch conversion and nullable helper admission | [Union transform carrier](../../../../codegen/tla/union_transform_carrier/README.md); call/declaration and name-only carrier counterexamples, with direct shape controls and generated HTTP round trips |
| Resolved defaults lowered into native physical layouts and finite nested expansion | [Default representation](../../../../codegen/tla/default_representation/README.md); bounded map/struct and recursive-expansion counterexamples, with three-transport compilation/runtime and codec controls |
| Custom String representation, selected default/supplied conversion and recursive failure propagation | [Custom representation](../../../../codegen/tla/custom_representation/README.md); literal, panic, dropped failure and stale graph counterexamples, with service/HTTP/JSON-RPC/gRPC compiled runtime controls |
| Tagged union envelope presence and semantic null admission | [Null admission](../../../../internal/uniongen/tla/null_admission/README.md); blanket rejection/acceptance counterexamples, with service, projected and transport decoding controls |
| Untagged branch validator physical context, nil guards and requiredness | [Union validation context](../../../../codegen/service/tla/union_validation_context/README.md); projected and ordinary declaration agreement, separate from recursive traversal |
| Collection element nullability before lossy conversion | [Collection decoding](../../../../http/codegen/tla/collection_decode/README.md); bounded decoding/conversion/view ordering and generated HTTP/WebSocket/request matrices |
| Preservation of absent fields before selected-view validation | [Projected presence](../../../../codegen/tla/projected_presence/README.md); bounded object/scalar panic and collection-presence controls, with generated HTTP/WebSocket tests |
| Concrete declaration/helper identity versus nullable occurrence storage | [Projected declaration identity](../../../../codegen/service/tla/projected_declaration_identity/README.md); illegal-identifier, first-occurrence parameter and erased-presence counterexamples, with service/view/HTTP collectors and generated transport/selected-view controls |
| Selected body logical origin and physical Go member identity | [Selected-body field identity](../../../../http/codegen/tla/selected_body_field/README.md); logical-only and physical-only counterexamples, mapped semantic keys and HTTP/JSON-RPC generated selectors |
| HTTP/JSON-RPC conversion identity and package declaration allocation | [Constructor ownership](../../../../http/codegen/tla/constructor_ownership/README.md); lexical-only, semantic-only, signature-only and type-dedup counterexamples, with generated compile and selected-field wire controls |
| Selected primitive body source/target storage and absent source conversion | [Response field storage](../../../../http/codegen/tla/response_field_storage/README.md); service-requiredness, blanket-pointer and unguarded-read counterexamples, with HTTP/JSON-RPC constructors and mounted HTTP controls |
| Shared response projection before unary and streaming framing | [Response projection](../../../../http/codegen/tla/response_projection/README.md); constructor-free selection leaks in three channels, named-empty endpoint binding, rejected codec/view/SSE shortcuts and mounted wire/client/notification controls |
| Selected-view nullable root, field and element carriers | [Selected-view carrier](../../../../codegen/service/tla/selected_view_carrier/README.md); ten rejected physical/admission/envelope/occurrence alternatives and 432 checked states, with complete generated HTTP/JSON-RPC runtime controls |
| View union allocation before conversion references | [View union names](../../../../codegen/service/tla/view_union_names/README.md) |
| Relocated type declarations, local reservations and union helpers | [External type names](../../../../codegen/tla/external_type_names/README.md) |
| Service-selected views, output validation and fault classification | [Result output boundary](../../../../codegen/service/tla/result_output/README.md); bounded legacy classification, unchecked-content and nil-element counterexamples |
| Generated service client, transport and interceptor stream operation agreement | [Client stream contract](../../../../codegen/service/tla/client_stream_contract/README.md); role/signature/cleanup counterexamples with typed SSE/WebSocket clients and ordinary gRPC controls |
| Client endpoint role and interception applicability | [Client interceptor dispatch](../../../../codegen/service/tla/client_interceptor_dispatch/README.md); mixed unary cast, lost payload and lost message counterexamples with typed and mounted clients and CLI role controls |
| Shared server/client interceptor accessor declaration coverage | [Interceptor accessor coverage](../../../../codegen/service/tla/interceptor_accessor_coverage/README.md); missing coverage, duplicate declarations and widened invocation counterexamples with generated typed-client and server controls |
| Stream-result attribute and accessor nominal identity | [Streaming result identity](../../../../codegen/service/tla/streaming_result_identity/README.md); disjoint-field rejection, partial-repair and nominal-cast counterexamples with mounted HTTP/JSON-RPC clients |
| Final client-streaming result interception and selected views | [Final result interception](../../../../codegen/service/tla/README.md) |
| Credential extraction before constraints and typed construction | [Bearer protocol boundary](../../../../security/tla/bearer/README.md); bounded valid-frame rejection control |
| Custom multipart accumulator, error-return signature and callback/validation barriers | [Multipart decode ownership](../../../../http/codegen/tla/multipart_decode/README.md); shared partials with wrapper-local control flow |
| HTTP captured status, Debug transparency and bounded body retention | [Response capture](../../../../observability/transport/tla/response_capture/README.md); informational/final/flush ordering, accepted bytes, stable commit metadata, opaque copying and explicit overflow omission |
| Client Debug passive response capture and exchange completion | [Client response observation](../../../../http/tla/client_debug_response/README.md); eager-read/error replacement, duplicate completion and wrong-exchange counterexamples; runtime tests cover redaction and concurrent Close |
| Immutable diagnostic credential projection | [Diagnostic redaction](../../../../observability/diagnostics/tla/redaction/README.md); heuristic-only, logical-name and borrowed-metadata counterexamples; generated wire roles and delayed completion tests |
| Parent-aware trace sampling | [Parent sampling](../../../../observability/otel/tla/parent_sampling/README.md); root-policy override counterexample, SDK composition checked against provider and HTTP propagation controls |
| Logger request scope independent of IDs | [Request scope](../../../../clue/log/tla/request_scope/README.md); root-sharing and copied-flush counterexamples, fresh request ownership across HTTP and both gRPC server paths |
| Shared logger buffer lineage and original entry outputs | [Buffer lineage](../../../../clue/log/tla/buffer_lineage/README.md); copied-buffer loss/duplicates and flush-time output selection counterexamples, paired with sibling serialization and request/Endpoint regressions |
| Logger configuration snapshots | [Logger options ownership](../../../../clue/log/tla/options_ownership/README.md); shared and shallow-copy counterexamples, detached snapshots, reused options and HTTP/gRPC Debug isolation |
| Debug request observation | [Request capture](../../../../observability/transport/tla/request_capture/README.md); passive downstream reads, original read/Close ownership, bounded preview and no eager drain |
| Multipart request source bytes, parser EOF and callback completion | [Multipart source budget](../../../../http/tla/multipart_source_budget/README.md); value-only and raw-only counterexamples, exact admission and mounted built-in/custom callback controls |
| Consumed response-body ownership, restoration and cleanup errors | [Response body lifecycle](../../../../http/tla/response_body/README.md); bounded ownership and error-preservation controls |
| Declared error properties across problem projection and client reconstruction | [Problem properties](../../../../http/tla/problem_properties/README.md); loss and status-inference counterexamples with full, selected and mapped transport controls |
| Built-in HTTP error public representation and projected field membership | [Problem response projection](../../../../http/codegen/tla/problem_response_projection/README.md); bounded full-body assumption counterexamples paired with generated scalar/object/header/cookie wire tests |
| Named URL field serialization, source admission and nested JSON carrier selection | [URL serialization](../../../../internal/httpencoding/tla/serialization/README.md); runtime/schema/import agreement and null-preservation counterexamples paired with generated form/query/import wire tests |
| HTTP response first-match priority and default placement | [Response order](../../../../http/codegen/internal/transportir/tla/response_order/README.md) |
| HTTP body wrapper copy/example identity and generated type names | [Body type names](../../../../http/codegen/tla/body_type_names/README.md) |
| HTTP type collection, validator eligibility and multipart cycle identity | [HTTP type identity](../../../../http/codegen/tla/type_identity/README.md) |
| HTTP union allocated identity and prepared-body package reachability | [HTTP union declarations](../../../../http/codegen/tla/union_declarations/README.md) |
| HTTP WebSocket payload null versus end-of-input framing | [WebSocket payload nullability](../../../../http/codegen/tla/README.md) |
| Protobuf endpoint/anonymous message name reservations | [Message names](../../../../grpc/codegen/tla/message_names/README.md) |
| Anonymous protobuf collection declaration contracts | [Wrapper contracts](../../../../grpc/codegen/tla/wrapper_contracts/README.md) |
| Protobuf collection wrapper contents and message presence | [Collection contents](../../../../grpc/codegen/tla/collection_contents/README.md) |
| Protobuf explicit names, collection wrappers, compatible declarations and recursive normalization | [Message declarations](../../../../grpc/codegen/tla/message_declarations/README.md) |
| Reachable protobuf field validation, cycles and opaque custom mappings | [Field tag traversal](../../../../expr/tla/grpc_field_tags/README.md) |
| Protobuf emitted field-number domain, normalized identity and nonwrapping union derivation | [Field number admission](../../../../expr/tla/field_number_admission/README.md); invalid/overflow/selection/publication counterexamples, with CLI rejection and unused-seed shape/generation controls |
| gRPC complete-source and recursive local presence admission | [Presence admission](../../../../expr/tla/grpc_presence_admission/README.md); placeholder ordering, shallow-local and over-broad Any counterexamples; direct recursive pairing and generated Any controls cover classifier limits |
| Shared protobuf helper error signatures and transitive propagation | [Helper fallibility](../../../../grpc/codegen/tla/helper_fallibility/README.md); root-inherited and partial-fix counterexamples, with both-order mixed helpers, recursion, generated compilation and Any error controls |
| Protobuf nil collection element admission before conversion | [Element admission](../../../../grpc/codegen/tla/element_admission/README.md); panic, guard-only wire loss and blanket-nil counterexamples, paired with physical-source, mixed-helper and mounted transport checks |
| Protobuf scalar presence, requiredness and defaults | [Scalar presence](../../../../grpc/codegen/tla/scalar_presence/README.md); bounded omission/zero checks, with real protobuf and generated decoder/stream tests |
| gRPC canonical binary metadata bindings before security admission and codegen | [Metadata bindings](../../../../expr/tla/grpc_metadata_bindings/README.md); raw-name, encoder-only, collision and late-credential counterexamples, with independent TCP peers |
| gRPC metadata absence-only defaults across request, headers, trailers and admission | [Metadata defaults](../../../../grpc/codegen/tla/metadata_defaults/README.md); current omission counterexample and required/supplied/malformed controls, with generated typed and mounted wire tests |
| gRPC named metadata element storage, codecs, constructors and CLI agreement | [Metadata elements](../../../../grpc/codegen/tla/metadata_elements/README.md); nominal-variable and partial-repair counterexamples, qualified service casts, inherited validation/defaults and real TCP controls |
| Map-key storage admission independent of ordinary type validation | [Map-key admission](../../../../expr/tla/map_key_admission/README.md); missing, type-global cached and primitive-only counterexamples, with evaluator rejection and generated comparable-key controls |
| Protobuf fields, getters, synthetic oneofs and map-entry wrapper names | [Field names](../../../../grpc/codegen/tla/field_names/README.md) |
| Generated import aliases, reciprocal local reservations and retained example bindings | [Import aliases](../../../../grpc/codegen/tla/import_aliases/README.md) |
| Conversion default admission from actual source absence and target materialization | [Default application](../../../../codegen/tla/default_application/README.md); zero-substitution, bytes-kind, nil-only and target-only counterexamples, paired with shared and protobuf conversion plus mounted CLI/body controls |
| CLI JSON decoding role, required admission and absence-only defaults | [CLI JSON input](../../../../http/codegen/tla/cli_json_input/README.md); bounded encoder-loss, zero-substitution and default-seeding counterexamples, with detached layout and mounted HTTP/JSON-RPC CLI controls |
| Shared CLI direct input selection by logical payload source | [CLI source selection](../../../../http/codegen/tla/cli_sources/README.md); append and printed-name counterexamples, with preserved body carriers, named/defaulted credential and repeated wire controls |
| Aggregate CLI import, usage and flag-set identifiers and their references | [CLI identifiers](../../../../codegen/cli/tla/identifiers/README.md); bounded allocation safety, with renderer reservations and per-server isolation checked in Go |
| Protobuf named-map oneof branches, including selected empty maps | [Map union branches](../../../../grpc/codegen/tla/map_union_branches/README.md) |
| JSON-RPC stream view selection and validation | [Stream/view decoding models](../../../../jsonrpc/codegen/tla/README.md) |

These local READMEs identify the generator seams, direct regressions and generated
compilation or round-trip checks that connect the abstractions to production.
Name-allocation assumptions are not proved merely by proving declaration lookup.

## OpenAPI contracts

- [YAML merge ownership](../../../../internal/openapiimport/tla/yaml_merge_ownership/README.md):
  shared-source mutation counterexample and immutable worker-view invariants;
  dependency race, metadata and rendering tests establish correspondence, while
  alias expansion correctness and arbitrary application mutation remain outside
  the bounded model.

- [Import contract comparison](../../../../cmd/loom/tla/import_contract_comparison/README.md):
  test-only retained reference ownership prevents occurrence loss and premature
  target pruning. Raw schema-position rewriting and exact numeric literal
  preservation remain Go regression obligations; public import/export admission
  and general JSON Schema equivalence are outside this bounded model.

- [Imported security credentials](../../../../internal/openapiimport/tla/security_credentials/README.md):
  bounded physical-slot/codec/component grouping retains all authentication roles
  and requirement structure. Actual import, generated authentication and OpenAPI
  tests establish concrete correspondence; partial root-inheritance projection
  is a separate tested obligation outside the model.

- [Imported recursive schema references](../../../../internal/openapiimport/tla/schema_references/README.md):
  bounded initializer graphs and per-occurrence constructor classification retain
  mandatory edges and lower eligible recursive references. The independent graph
  oracle, evaluated JSON/YAML constructor cases and generated schema/build checks
  establish concrete correspondence; the models do not prove Go initialization,
  naming or schema extraction.

- [Pattern projection](../../../../http/codegen/openapi/internal/ir/lean/pattern_projection/README.md):
  finite import/export anchor/whitespace witnesses and translated acceptance tables;
  Go/Node tests establish actual engine agreement for the admitted boundary.

- [Document identities and explicit component claims](../../../../http/codegen/openapi/internal/ir/tla/document_identity/README.md):
  finite duplicate operation, equivalent path-template and conflicting media
  name counterexamples; accepted controls retain multiple methods on one path
  and equal component claims. Concrete extraction, serialization and Go
  refinement remain test obligations.

- [Strict import schema normalization](../../../../internal/openapiimport/lean/schema_normalization/README.md):
  finite acceptance counterexamples for boolean schemas, typeless properties,
  constrained null compositions and dialect-dependent nullability; admitted
  candidate forms preserve the modeled acceptance domain. Go tests establish
  the concrete admission and rendering correspondence within the stated scope.

- [Responses sharing a status](../../../../http/codegen/openapi/internal/ir/tla/response_media/README.md):
  bounded media/schema preservation, declaration-order independence and rejection
  of incompatible metadata; rendered and runtime tests cover concrete extraction.

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
| Artifact write intent, atomic create-only installation, rollback ownership and gen-tree admission | [Artifact write intent](../../../../cmd/loom/tla/artifact_write_intent/README.md); bounded intent-loss/race controls paired with real filesystem and registered-plugin CLI tests |
| Service-error merge and history snapshot ownership | [Error ownership](../../../../pkg/tla/error_ownership/README.md); bounded aliasing counterexamples, with repeated merges and protobuf reconstruction checked in Go |
| Public message ownership, history masking, diagnostic exclusion, transparent wrappers and independent joins | [Public disclosure](../../../../pkg/tla/error_disclosure/README.md); finite owner/message model, with HTTP/gRPC/JSON-RPC unary/SSE/WebSocket wire tests |
| gRPC aggregate error status and detail ownership | [Error contracts](../../../../grpc/tla/error_contract/README.md); bounded branch-order and complete-failure checks, with wrappers and generated unary/streaming mappings tested in Go |
| Nearest raw forwarding boundary and prefix independence | [Forwarded client trust](../../../../http/tla/forwarded_client/README.md); bounded chains, with IP parsing and repeated-header grouping checked in Go |
| Authentication alternatives, optional credential extraction and failed-context isolation | [Credential alternatives](../../../../internal/securitygen/tla/credential_alternatives/README.md); finite extraction/control model, with actual framing and HTTP/JSON-RPC/gRPC wire regressions |
| gRPC pre-frame admission, complete application input, original cancellation and middleware rechecks; ordered AND/OR contexts and failed-branch reset | [Stream admission](../../../../security/tla/stream_admission/README.md); positive and deliberate negative models paired with runtime and generated transport regressions |
| Scalar query/header/cookie absence, defaults and conversion admission | [Scalar binding presence](../../../../http/codegen/tla/scalar_binding_presence/README.md); supplied-empty counterexample, shared decoder and mounted HTTP/JSON-RPC controls |
| Basic-auth header presence, component projection and required rejection | [Basic presence](../../../../http/codegen/tla/basic_auth/README.md); bounded required/optional and absent/empty/value combinations |
| Escaped route literals and split-before-unescape path arrays | [Path decoding](../../../../http/tla/path_decoding/README.md); bounded operation-order checks, with raw capture preservation and actual dispatch checked in Go |
| gRPC operation contexts, bounded native I/O, callback arbitration, half-close, data ownership and panic recovery | [Message contexts](../../../../grpc/tla/message_context/README.md); current and rejected lifetime/ownership models, including terminal half-close admission, paired with native bufconn and generated mounted cancellation tests |
| gRPC opening-send EOF and final status ownership | [Initial send model](../../../../grpc/codegen/tla/initial_send/README.md); bounded stream retention and receive-completion checks |
| Telemetry provider construction, global publication and reverse cleanup attempts | [Construction ownership](../../../../observability/otel/tla/construction/README.md); early-publication, missing-cleanup and wrong-order counterexamples; actual SDK failure and cleanup context/error tests state reclamation limits |
| gRPC tracing-test span publication observation | [Span completion](../../../../grpc/middleware/otel/tla/span_completion/README.md); RPC-return and notify-before-record counterexamples, explicit completion notification, gated runtime schedule and bounded deadline |
| WebSocket framing and expanded-message limits, overflow release and accepted bounds | [Message admission](../../../../http/tla/websocket_message_limit/README.md); bounded framing-only counterexample paired with compressed real-socket and generated client/server tests |
| WebSocket sole-reader ownership, bounded byte/count admission, attachment and terminal release | [Connection lifetime](../../../../http/tla/websocket_lifetime/README.md); empty-frame overload, blocked-writer and peer-acknowledgement-before-publication counterexamples, with synchronized runtime and generated HTTP/JSON-RPC tests |
| SSE complete-event dispatch, EOF tail discard and iterator lifecycle | [SSE framing](../../../../http/tla/sse_framing/README.md); bounded framing and read/stop exclusion models paired with shared-reader and generated-client tests |
| JSON-RPC mount/Use order and configured handler dispatch | [Handler dispatch](../../../../jsonrpc/codegen/tla/handler_dispatch/README.md); configuration-before-requests contract with mount-order counterexample |
| JSON-RPC WebSocket request failure versus connection failure | [Request completion](../../../../jsonrpc/tla/notification_completion/README.md); streaming invalid-notification and blanket-suppression counterexamples; mounted same-connection barriers and transport-error identity controls |
| JSON-RPC WebSocket credential location and request-role survival | [Effective security admission](../../../../expr/tla/jsonrpc_ws_admission/README.md); raw-map, location-only and blanket-rejection counterexamples, with DSL and mounted message-authentication controls |
| JSON-RPC WebSocket response routing, client closure/redial and closure error identity | [JSON-RPC connection models](../../../../jsonrpc/tla/README.md) |
| Redis job ownership, fencing/resume, requeue replies, event acknowledgments, join reconciliation and shutdown watcher startup | [Pulse pool models](../../../../pulse/pool/tla/README.md) and their linked ownership design/action map |
| Reader/Sink subscription admission and Reader worker startup after shutdown | [Subscriber admission](../../../../pulse/streaming/tla/subscriber_admission/README.md); bounded late-channel/worker counterexamples and concurrent admission tests; excludes I/O termination |
| Sink first-subscription delivery activation | [Sink activation](../../../../pulse/streaming/tla/sink_activation/README.md); eager read and partial claim counterexamples, one-time read/claim activation with keepalive retained; excludes all-unsubscribed replay and I/O termination |
| Reader registration identity across admission, reply, fanout and replacement | [Reader registration model](../../../../pulse/streaming/tla/reader_registration/README.md); stale-cursor counterexamples and synchronized replacement/close tests, preserving admitted delivery |
| Reader newest snapshots and duplicate/close publication rechecks | [Reader cursor model](../../../../pulse/streaming/tla/reader_cursor/README.md); one-time boundaries and atomic publication, excluding stale-registration replies and shutdown termination |
| Sink missing-group recovery and retained start selectors | [Group recovery](../../../../pulse/streaming/tla/group_recovery/README.md); read/rotation hot-loop and stale-registration counterexamples, serialized repair and finite-failure recovery; excludes deleted events and PEL |
| Sink filter identity, heartbeat refresh versus retirement, and pending work before group deletion | [Filter membership and retirement](../../../../pulse/streaming/tla/filter_membership/README.md); stale-snapshot and split-recheck counterexamples, membership CAS and atomic pending-entry guard, with synchronized Redis regressions |
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

The [discriminator admission model](../../../../http/codegen/openapi/internal/ir/lean/discriminator_metadata/README.md)
records optional/fallback admission and 3.1 projection. Go tests cover reference
resolution and graph retention outside its Boolean assumptions.
| JSON-RPC envelope ID source, presence and validation ordering | [Envelope ID admission](../../../../http/codegen/tla/jsonrpc_id_admission/README.md); params-first and late-handler counterexamples, shared carrier/CLI validation and mounted unary/SSE/WebSocket controls |
