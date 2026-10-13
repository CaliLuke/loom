---
name: loom-framework
description: Develop and maintain the Loom framework repository itself. Use for changes to DSL implementation, expr, codegen, HTTP/gRPC/JSON-RPC transports, OpenAPI generation, framework runtime packages, fixtures, internal architecture, or contributor workflows. Do not use for ordinary application-level Loom adoption.
---

# Maintain the Loom Framework

Use this skill for work inside the Loom framework repository: implementing or
reviewing DSL behavior, expression evaluation, generators, transport runtimes,
OpenAPI output, framework tests, fixtures, and contributor tooling.

Do not use this skill for ordinary consumer work in `design/`, `gen/`, service
implementations, or application bootstrap. Use the `loom` skill for those
tasks. This skill also owns the stronger delivery workflow for new framework
capabilities; there is no separate capability-maintenance skill.

## Start Here

1. Read the repository `AGENTS.md` completely.
2. Identify the owning layer before editing:
   - `dsl/` for public authoring functions
   - `expr/` for evaluated design semantics and validation
   - `codegen/` for shared generated Go models
   - `http/`, `grpc/`, or `jsonrpc/` for transport-specific behavior
   - `http/codegen/openapi/internal/ir` for OpenAPI analysis and contract
     decisions
   - `http/codegen/openapi/v3` for versioned OpenAPI rendering
   - `vet/` for evaluated-design and consuming-module adoption diagnostics
3. Inspect nearby direct tests and meaningful fixtures.
4. Read the relevant public guide and the consumer `loom` skill only when the
   task changes how users design or operate a Loom service.
5. Find the owning proof or model in the [formal model index](references/formal-models.md).
   Read its assumptions, implementation correspondence and expected failures
   before changing the behavior it describes.

Fix the root framework boundary. Do not patch generated `gen/` output or add an
application-facing workaround when the framework owns the behavior.

## Adding or Changing Framework Capabilities

Apply this section when adding public DSL surface, generator or transport
behavior, OpenAPI or JSON Schema features, auth/session semantics, request or
response decoding, or another framework behavior intended to remove
application glue.

Before implementing, identify:

1. the repeated application workaround or concrete risk
2. the generic framework behavior that should replace it
3. a real consuming application or contract that proves the need
4. the framework layer that owns the behavior
5. what policy must remain application-owned

If those boundaries are unclear, do not add framework surface yet. Prefer the
narrowest behavior that removes the actual workaround, and do not special-case
one application's protocol flow when a generic transport or contract behavior
is the real fix.

A capability is complete only when:

- implementation lives at the correct ownership layer
- direct, generated-output, regression, and broader transport tests cover the
  relevant behavior
- coverage includes normal, edge, invalid, ambiguous, and still-rejected cases
  where applicable
- public docs and the consumer `loom` skill describe changed application usage
- internal architecture or workflow changes are recorded in this skill
- any live roadmap item reflects the delivered state
- the change is committed and pushed unless the user asks otherwise

For decoding capabilities, cover successful decoding, invalid input,
validation after decoding, and removal of the old custom workaround. For
contract output, pair structural assertions with rendered output and parser or
consumer validation.

## Architecture Boundaries

- `http/codegen.buildCLIArgumentPlan` owns selected endpoint-bearing service order
  and the parser/example dependency sequence for HTTP and JSON-RPC. Keep exact
  service and endpoint identity through command-name allocation; project each
  renderer's own aliases from that plan. Do not regroup multipart and interceptor
  arguments independently or infer identity from rendered names/configurer nils.

- `expr.validateErrorNames` admits authored API, service and method error lists
  before finalization. Exact wire names are unique within each original scope;
  narrower overrides, sibling scopes and distinct Go-normalized names remain
  legal. Preserve the existing per-error checks and method/service merge order.
  Do not deduplicate lists or validate only merged method errors, which can hide
  unused or overridden duplicates. Keep the error declaration admission model
  and evaluator/CLI rejection controls aligned.

- `codegen/service.serviceSymbols` records fixed and method-derived package
  declarations before nominal types are collected. Its private
  `errorConstructorName` owner reserves emitted factories by exact wire error
  identity in the same cached NameScope, before nominal types. Declaration and
  method error records retrieve that identity; custom errors with no emitted
  factory keep an empty factory name. Do not allocate factories per reference or
  infer their spelling in transports. Method/member and stream
  adapter names have separate scopes; do not put package reservations back into
  method naming. Reuse effective security, HTTP/JSON-RPC and interceptor analysis
  for conditional declarations. Render endpoint factories, carriers, SSE events
  and method-derived interceptor wrappers from recorded identities, including
  HTTP/gRPC CLI references. Keep canonical external type names and the shared
  nominal NameScope cache. The package-symbol models and bounded artifact matrix
  cover declaration ownership; independent interceptor/protobuf/member families
  remain separate. A corrected service method identity also changes the gRPC RPC
  name derived from it, so retain descriptor and regeneration migration coverage.

- `observability/otel.New` stages all enabled providers and their shutdown
  callbacks before publishing SDK globals. Failed construction uses
  `rollbackProviders` with a bounded uncanceled context; both it and successful
  Runtime.Shutdown consume the same reverse `shutdownProviders` owner. Attempt
  all callbacks and join errors. Disabled sections leave corresponding globals
  untouched; propagation updates only when traces are enabled. Keep the construction model and actual failed-bootstrap plus cleanup
  context/order/error tests aligned; do not claim atomic multi-global publication
  or forced reclamation from uncooperative callbacks.

- `observability/otel.newTraceProvider` composes SDK ParentBased around the
  existing normalized TraceIDRatioBased root policy. Preserve both local and
  remote parent decisions; invalid parents and explicit new roots use the root
  policy. Do not recreate sampling decisions in HTTP/gRPC wrappers. Keep the
  parent-sampling model, direct provider and actual HTTP propagation tests aligned.

- `clue/log` formatters use the standard JSON encoder for syntax. Keep ordered
  fields and duplicate names through `jsonEntry`; do not convert entries to maps.
  `jsontext.Float`/`Float32` own scalar nonfinite presentation. Complex codec
  failures remain per-value diagnostic strings. Contain rejected array starts and
  raw values using the standard encoder's unchanged-state error contract; do not
  impose a separate nesting counter or abort siblings at its depth limit.
  Text and terminal names/values
  pass through the same presentation escaping boundary, including fmt fallbacks.
  Keep direct control-character, invalid UTF-8, nonfinite, codec-error and
  ordinary exact-output tests plus bounded formatter benchmarks aligned.
- `clue/log.Context` owns logger configuration publication. Under the existing
  logger lock, detach options and their slice storage before applying updates;
  `With` may share the immutable snapshot. Preserve the logger's buffer identity
  and fields. Reusable `WithOutputs` options detach their supplied slice on each
  application and never write it. Writers, callbacks and field values remain
  application-owned. Keep the options ownership model under `clue/log/tla/`
  aligned with direct configuration, concurrent reuse and HTTP/gRPC Debug tests.
  Request-scope creation and buffer lineage are separate ownership contracts.
- `clue/log.logBuffer` owns queue, flush state and output serialization for one
  scope shared by With descendants. Fresh Context/requestContext scopes allocate
  a new owner. Queued entries retain immutable output settings from admission;
  logger configuration changes apply only to later entries. Always acquire logger
  before buffer locks; never acquire a logger from the buffer owner. Print bypass
  does not flush the queue. Keep lineage/output-selection models and transport,
  configuration and sibling-serialization regressions aligned. Application values
  and callbacks remain caller-owned; reentrant logging is outside the contract.
- `clue/log.requestContext` creates each HTTP/unary/streaming server logging
  scope from a locked root configuration/field snapshot. Start an empty buffer
  with fresh flush state and evaluate buffering policy using incoming request
  context. Optional ID metadata is supplied as initial fields, independent of
  scope creation; do not use With to clone a request's lifetime. Keep the request
  scope model and both ID/access-log modes aligned. Filtered HTTP bypass and
  application-owned context lifetime remain unchanged.

- Generated artifacts retain their write intent from rendering to live
  installation through `codegen.Artifact`. `internal/artifactpolicy` owns
  admission, including rejection of create-only files inside `gen/`, stable live
  aliases into it, and case variants of its reserved top-level name. External
  create-only writes use exclusive creation or atomic no-replace installation;
  skipped destinations are never acquired for rollback or reported as installed.
  Preserve intent across the helper JSON protocol. `Generator.Run` returns raw
  bytes shared by generation and vet, whose callers decode their own result.
  Keep the model under `cmd/loom/tla/artifact_write_intent` aligned with real
  plugin/CLI, creator-race and rollback regressions.
- `pkg.ValidateFormat` owns hostname validation: complete ASCII label syntax,
  63-byte labels, 253-byte text excluding an optional terminal root dot. Keep
  this shared by generated transports and design validation. The existing
  gojsonschema v1.2.0 checker rejects root dots and allows overlong 254/255-byte
  unrooted text; IDNA conversion has different Unicode/Punycode semantics.
  Use the bounded standard-library regex here instead of adding either policy
  as a runtime dependency. Host existence and IDNA processing stay caller-owned.
- `internal/encodingmeta.ResolveJSONField` owns authored tag normalization and
  omission resolved against actual field storage. `expr.ServiceFieldLayout` is
  shared by service declarations and canonical Extend identity; run identity
  checks after inherited defaults finalize while retaining authored provenance.
  HTTP `representation.ResolveField` owns decoder wrappers and pointer choices;
  declarations and value projection consume its same resolved JSON action.
  Restore registered named layouts before selecting fields. Automatic omission
  preserves supplied empties and native zeros; full tags keep their intent,
  including both omission predicates. Opaque replacements keep their codec policy.
  Do not infer absence from arbitrary IsZero methods or restore a transport-only
  tag merger. Keep the [omission model](../../../codegen/tla/json_omission/README.md),
  carrier ledger, generated HTTP/RPC wire and OpenAPI example controls aligned.
- Selected body member identity is resolved once by
  `http/codegen.selectBodyField`: logical origin and actual object key retain
  design semantics, while `GoifyAtt` supplies the declared Go member. Request,
  response, error and detached body constructors consume that binding. Do not
  reconstruct physical selectors from logical names in downstream renderers.
  Keep the [selected-body identity model](../../../http/codegen/tla/selected_body_field/README.md)
  and renamed/mapped/whole-body generated controls aligned. Physical storage is
  independently derived from the canonical declaration layout below.
- Selected primitive body fields retain their enclosing declaration storage
  through `http/codegen.serviceFieldPrimitivePointer`, derived from
  `expr.ServiceFieldLayout` with the actual service/view context. Client result
  constructors use it for addressing; request extraction retains its admission
  guard, and response conversion preserves native source pointer absence in its
  body reference and guarded initializer. Nullable/custom-presence wrappers are
  already values; do not address or dereference them as native pointers. Keep
  [response field storage](../../../http/codegen/tla/response_field_storage/README.md)
  aligned with generated constructor and mounted nil/empty/value controls.
- `http/codegen.ResponseEncodePlan.BodyValues` owns complete response values by
  body/view variant, including selected fields that need no constructor. HTTP
  encoders, HTTP WebSocket final results and JSON-RPC framing consume these same
  expressions. JSON-RPC endpoint result binding also derives from the effective
  body's presence, while metadata capture and WebSocket forwarding retain their
  sources. A named empty result is not evidence of a body. JSON-RPC binds the
  canonical projection source in a private rendering scope;
  do not restore constructor-only/raw-result fallbacks. Preserve metadata capture
  and configured encoder invocation counts. Mixed SSE events retain their
  distinct StreamingResult source. Keep the
  [response projection model](../../../http/codegen/tla/response_projection/README.md)
  and mounted selected-body, notification, codec and view controls aligned.
- JSON-RPC request IDs bind once in `expr.JSONRPCIDBinding` before body
  projection. Transport IR and `http/codegen.JSONRPCIDData` retain source
  identity and physical storage. Decode the envelope through the shared field
  validation/conversion owners before constructing the service payload; never
  inject IDs in handlers. Params projection excludes the field across HTTP,
  SSE and WebSocket clients, including ID-only requests and CLI builders.
  `RawRequest.HasID`/`ID` remain protocol correlation, independent of service
  defaults. Client notification handling consumes the request encoder's decision
  without running custom codecs again. Keep the envelope admission model and
  mounted carrier/streaming tests aligned.
- Design DSL is the source of truth; evaluated semantics belong in `expr`.
- `RootExpr.Prepare` composes `Extend` after promoted union naming and before
  dependent transport preparation or validation. Preserve authored overlays and
  source provenance during DSL execution; compose detached occurrence graphs
  through the existing effective-constraint owner. Finalizers and consumers
  must not reselect inherited fields or simulate a separate base precedence.
  Defaults compare resolved values against the complete effective contract;
  `internal/encodingmeta` owns the shared custom-type representation tuple,
  JSON tag normalization and explicit-presence override facts. Composition and
  codegen consume those owners; JSON identity/options compare under effective
  containing requiredness, including retained Extend parent constraints. Named
  field recursion is valid; reject inheritance cycles separately and detach
  snapshots only after referenced effective shapes are complete.
  Keep `expr/tla/effective_design` and its ownership/transport regressions aligned.
- Effective named-type constraints have one immutable `expr` owner per value
  occurrence. Derived enums refine ancestor enums, defaults are selected then
  checked against the complete effective contract, numeric and length bounds
  conjoin, pattern and format clauses conjoin in current-to-base order with
  exact duplicates retaining current provenance, and required fields union by
  finalized member identity. `EffectiveValidation` retains original provenance;
  `Lowered` is a detached predicate-equivalent physical carrier whose later
  query derives physical provenance. Lowering stores selected enum membership
  as clauses rather than re-authoring `Values`; `EnumCandidates` separately
  reports filtered representatives and distinguishes absence from a present
  empty domain. Shared `validationDomainErrors`/`validateEffectiveDomain` admission
  checks every authored layer before tighter-bound folding can hide negative
  lengths, then rejects contradictory effective numeric/length endpoints. Local
  ValidationExpr, named ancestry and Extend use this owner; do not reintroduce
  transport/composition interval checks. Endpoint admission does not solve
  machine-carrier representability or arbitrary pattern satisfiability. Keep
  `expr/tla/constraint_domain` and direct/DSL checks aligned.
  Renderers may retain authored overlays for
  schema composition, but must not select semantic precedence independently.
  `Reference` remains object-template copying rather than named-type ancestry.
- Mapped attribute preparation retains the eliminated named declaration edge
  before existing constraint parents. Keep inherited enums at their declaring
  occurrence; never copy a selected public validation carrier onto a refined
  body as newly authored constraints. Controlled ancestor-enum projection matches
  logical attribute names while preserving target wire spelling. OpenAPI enum
  emission and schema identity consume the same checked annotation owner, even
  without local validation. Raw example-seed hashing remains structural. Keep
  `expr/tla/prepared_enum_provenance` and generated body/spec controls aligned.
- `internal/exactnum.Number` owns immutable exact numeric values across DSL,
  source import, expressions, native thresholds and JSON Schema. Preserve native
  float values and original JSON/YAML numeric kinds; source adapters own grammar,
  aliases and merge precedence before handing off exact values. Consumers must
  not round through Float64 or reinterpret a number's textual spelling. Generated
  validators derive directed thresholds for their concrete native type.
  Required schema defaults/enums consume checked value-plan projections; defaults
  use the final contract, while enum clauses retain their declaring occurrence
  through nested refinement. Keep known builtin checks even beside opaque codecs.
  The value-contract Lean model and extraction/conformance tests state the limits
  of the proof; cross-language token checks establish exact wire preservation.
- `expr.primitiveValue` owns primitive source admission and normalization for
  both `IsCompatible` and typed value resolution. Integer values must fit the
  declared carrier using exact limits; preserve concrete source-type eligibility
  and existing float literal rounding/underflow. Do not add renderer range guards
  or reject redundant numeric predicates merely because the bound is outside the
  carrier. Required defaults/enums recurse through the value owner; optional
  examples retain their existing partial-shape contract.
- Numeric example synthesis selects the concrete representation with `unalias`
  while retaining the occurrence's effective bounds. Named integer and Float32
  aliases must not enter the Float64 path merely because their outer kind is
  `UserTypeKind`. Keep this in `expr.byMinMax`; collection construction must not
  coerce wrongly typed examples. The direct numeric-alias tests cover all numeric
  kinds, inherited/occurrence bounds, exact and exclusive bounds, and typed arrays.
- `TransformAttrs.DefaultForField` owns conversion default admission: selected
  target default plus actual source absence capability and target materialization
  timing. Go and protobuf emitters consume the same record. Native scalar zeros
  are supplied; nil-capable storage and absent Optional/Nullable can default.
  Metadata storage overrides precede semantic kinds; pointer targets retain
  absence for selected-view validation. Required admission precedes conversion.
  Keep the [default application model](../../../codegen/tla/default_application/README.md)
  aligned with mounted CLI/body, protobuf bytes, custom storage and view controls.
- `AttributeContext.DefaultValue` owns native default materialization from the
  resolved `expr` value graph and target field layout. Shared and protobuf
  transforms use it after their existing application gates. Preserve member
  identity, registered named layouts and explicit custom/raw codec boundaries;
  reject authored members without native storage instead of discarding them.
  Nested default expansion tracks active value identities and rejects cycles.
  Keep the default-representation models and generated transport controls aligned.
- `expr.valueViewPlan.project` preserves intrinsic result declaration Nullable
  independently of copied occurrence Nullable and selected-view requiredness.
  Keep declaration cache identity keyed by source and selected view; a nullable
  field must not promote its shared declaration. The
  [projection nullability model](../../../expr/tla/project_nullability/README.md)
  covers this boundary; generated selected-view physical carriers remain a
  separate service-codegen obligation.
- Shared validation distinguishes complete named definitions from occurrences.
  `NamedValidationFunction` resolves the callable identifier, physical carrier
  and static field mode; consumers must use that same owner for declarations and
  calls. Preserve physical naming metadata at the occurrence edge while deriving
  parameter presence from the intrinsic named declaration; local Nullable and
  predicates must not redefine it. Keep the physical identity model and actual
  shared/protobuf scope regressions aligned. Named validator eligibility must
  restore registered layout at every named boundary and visit shared attributes
  separately under distinct physical modes. Sum-union eligibility and rendering
  share branch context and structural pointer presence, including branches with
  no authored predicates. Keep `codegen/tla/validation_layout` aligned with the
  nested-union, optional-view and both-order layout regressions. Intrinsically nullable named types
  retain their wrapper when called,
  while occurrence predicates inspect the unwrapped value. Anonymous object and
  enum guards use the same occurrence carrier fact without clearing child field
  layout; reset that fact when traversing a child. Keep the validation-carrier
  model and generated anonymous-object controls aligned. Recursive occurrences
  call complete definitions with their diagnostic path, never copy an in-progress
  buffer or impose an artificial traversal depth. Keep the recursive-validation
  model and finite generated-tree/nullable-carrier controls aligned.
- `pulse/streaming.prepareReaderCursor` owns one-time newest snapshots for Reader
  construction and stream addition. Keep the authored start mode separate from
  concrete per-stream cursors. Perform lookup before publication, outside the
  Reader mutex, then recheck duplicate/closing state before attachment. Preserve
  explicit positions without lookup and propagate lookup failures without
  changing active reads. Registration identity and shutdown termination remain
  separate obligations; the cursor snapshot model does not prove either.
- Reader and Sink subscriber admission uses `registerSubscriber` under the
  owner lifecycle lock. Closing or closed owners return an already-closed
  channel; only an admitted Reader subscriber can start the worker. Keep this
  admission rule separate from bounded I/O and concurrent Close joining.
- Reader registrations own stream identity and cursor together. Snapshot ordered
  registration references and detached Redis arguments at read admission, retain
  those references through reply/fanout, and advance only the identical installed
  registration under the Reader lock. Remove/re-add cannot receive an old read's
  cursor. Already-admitted events may still be delivered; this is not a drain
  guarantee or a shutdown-termination fix. Keep the reader-registration model
  and synchronized reply/fanout regressions aligned.
- Shared analysis belongs in a shared IR rather than being independently
  rediscovered by transport renderers.
- OpenAPI response accumulation belongs in `openapi/internal/ir`: collect every
  alternative before projecting one response per status. Merge media maps and
  schema alternatives, intersect header requiredness, and reject conflicting
  metadata rather than selecting the last declaration. Runtime response order
  and individual response contract cases remain transport-owned. See the
  [response media model](../../../http/codegen/openapi/internal/ir/tla/response_media/README.md).
- HTTP and JSON-RPC body declarations derive from registered prepared body roots.
  Register server response roots after view projection; declaration membership
  and branch validator requirements consume that same graph. Allocate each union
  once by its existing physical Go identity, then retain separate client/server
  membership and visited sets. Discover client roots in established endpoint order
  before server-only roots to preserve allocated names. Never emit a client-only
  union in a server package or add unreachable branches to satisfy it. Keep
  `http/codegen/tla/union_declarations` and its projection controls aligned.
- Shared sum-union validation derives selected-branch presence from the physical
  scope reference used by declarations. Reset the branch's concrete-value fact
  without discarding its child field layout: pointer storage requires a value,
  nil collections still run predicates, and explicit nullable carriers retain
  their presence semantics. Pointer presence itself requires a validation case
  even when the branch has no value predicates. Never infer representation from emitted validation
  text. Keep `codegen/tla/union_branch_carrier` and the direct/generated carrier
  regressions aligned; validator eligibility and JSON decoding remain separate.
- Shared union transformation admits direct named-object helpers only for
  concrete source and target carriers. Nullable branches must reach attribute
  dispatch first, so the existing presence owner unwraps the value, calls the
  concrete helper and preserves absent/null/value states. Helper collection and
  invocation must agree on physical carriers; a name-only repair is insufficient.
  Keep `codegen/tla/union_transform_carrier` and direct/generated controls aligned.
- `internal/uniongen` owns the common Go union declaration and codec emitter.
  Service and HTTP analysis supply that model; service-only validation helpers
  stay in service analysis. Keep untagged occurrence matching in
  `internal/unionjson`, and do not fork emitted union methods by transport.
  Tagged branch JSON admission consumes the `expr.AllowsNull` fact carried in
  `uniongen.Field`; a value property is mandatory even when it permits null.
  Service, projected and HTTP analysis must retain that semantic fact rather
  than infer it from physical type names. Keep the null-admission model under
  `internal/uniongen/tla/null_admission` and generated wire controls aligned.
- Put design-semantic vet rules over the evaluated `expr` graph. Use Go source
  analysis only for module adoption facts that the graph cannot contain. A
  source analyzer must report when an active target-module package lacks
  complete syntax or type information, without repeating usable-package or
  dependency diagnostics. Route conflict analysis must use exact typed
  method/path constants from active source files and skip dynamic expressions
  rather than infer speculative runtime overlap.
- Classify heuristic vet rules as warnings and provide a scoped suppression.
- Keep `untyped-semantic-attribute` conservative: never warn on `Any` alone;
  allow name-only inference only for timestamp-style names such as `*_at`,
  require explicit UUID evidence for IDs, and ignore ambiguous descriptions
  that describe multiple possible shapes.
- Keep `service-not-mounted` opt-in through API metadata. Scan typed `Mount`
  calls only in the configured consumer packages, count calls in conditional
  branches, support multiple packages, and let service metadata suppress an
  intentional omission. Do not infer runtime reachability or enable the rule
  for library modules that omit the metadata.
- Fingerprint the evaluated design and output-affecting generation context in
  `gen/loom.json`. The fingerprint must ignore runtime-only evaluator state,
  all `loom:vet:*` metadata, timestamps, and filesystem paths, and must
  serialize maps deterministically. `loom vet` reports design skew separately
  from Loom version skew.
- Keep helpers package-private or under an `internal` package when only one
  codegen area needs them.
- `internal/enumvalue` normalizes already canonical builtin scalars without
  reconstructing their enum occurrence. Normalization is not admission;
  validation remains with `expr`. Do not cache schema fingerprints across
  authored occurrences or bypass value resolution for collections and codecs.
- CLI JSON input has a decoding role independent of client package placement.
  `http/codegen/service_data_json_input.go` owns its detached carrier graph,
  declaration closure, validation and conversion for HTTP and JSON-RPC, including
  bare collection flags. Use existing body layout, NameScope and union validator
  owners; keep encoder declarations unchanged. Admit required members before
  defaults and preserve nullability/custom codecs through the JSON decoder.
  Keep the [CLI input model](../../../http/codegen/tla/cli_json_input/README.md)
  aligned with generated constructor and mounted CLI tests.
- CLI helpers live under the design module. Resolve the caller's workspace
  before loading and compiling the helper, and share that environment between
  both steps. Workspace mode must not receive module-only build flags; isolated
  vet module files remain the non-workspace path.
- Derive identifiers and generated directory names from design names with
  `internal/naming`. Generators call `naming.ServerDir` and
  `naming.ServiceDir` directly, the public `codegen` case functions such as
  `Goify` and `SnakeCase` and `expr.Title` delegate to it, and `expr` uses it
  to reject servers or services whose directories collide, ignoring case,
  and interceptors of one service whose `naming.Goify` names collide.
  `naming.GoifyAttribute` owns field-name overrides for both `codegen.GoifyAtt`
  and service-field collision validation. Check service data separately from
  transport mapping objects, whose names may legitimately collide as Go locals.
  Change the naming rules there only, so validation cannot drift from the
  generated paths. `naming.CLIDir` and `naming.TransportServiceDirs` record
  the client CLI tree and the transport packages of a service, which `expr`
  checks for a `cli` service; `TestCLIServiceTransportDirs` in
  `codegen/generator` fails when the generated layout drifts from them.
- Allocate aggregate CLI imports, usage functions, flag sets and flag variables in
  their shared output scope. Reserve fixed imports and parser locals, and carry
  allocated names to references instead of recomputing suffixes. Keep allocation
  data local to each server file. Generated service packages must be importable;
  the canonical service package allocator reserves `main` while preserving the
  authored service name and directory. See the [CLI identifier model](../../../codegen/cli/tla/identifiers/README.md).
- Resolve imported package identifiers with `internal/naming.ImportName` in
  import pruning, alias allocation and generator name reservation. Explicit aliases
  take precedence; unusual package declarations require an explicit alias. Keep
  file-specific local-name reservations in the owning generator.
- Use NameScope helpers (`GoTypeRef`, `GoFullTypeRef`, `GoTypeName`) for emitted
  Go type references. Never construct type syntax by string concatenation.
- Qualified collection references consume the same physical name owner as singular
  references: preserve allocated service identifiers and external declaration
  names after package selection. Do not reconstruct references with `Goify`. Keep
  collision, reservation, alias and presence controls paired with the bounded
  `codegen/tla/collection_identity` model and generated round trips.
- Shared default literal construction uses the target attribute and NameScope
  recursively for arrays and maps, including named outer containers and located
  elements. Ordinary assignments and explicit-presence conversions consume this
  same owner; never render a collection from the authored Go container type or
  maintain a separate alias formatter. Preserve typed nil versus empty literals,
  raw-JSON key/value representation and the existing default-application gates.
- gRPC metadata decoding owns one presence decision in
  `renderGRPCMetadataDecode`, shared by requests, response headers/trailers and
  early stream admission. `extractMetadata` resolves effective defaults into
  typed physical literals through `AttributeContext.DefaultValue`; constructors
  retain named service conversions. Default only absent optional entries, never
  supplied zero/empty or malformed text; required omission remains invalid.
  Keep `grpc/codegen/tla/metadata_defaults` and generated wire controls aligned.
  Repeated text entries retain their codec; do not claim empty-list presence.
- Let Loom determine pointer/value semantics except at explicit transport
  validation boundaries.
- Collapse pass-through wrappers that add no behavior.
- Accumulate generated source with standard-library `strings.Builder` and
  formatted writes. Pass populated builders by pointer; never copy them.
- Keep generated transport code declarative. Move stable protocol execution to
  handwritten runtime packages, and pass typed handlers or callbacks from
  generated code without broad reflection.
- Use `codegen.Section`, `codegen.JenniferSection`, and `codegen.RawSection` for
  Loom-owned Go generation. Use neutral `.tmpl` assets only for non-Go output
  that genuinely benefits from templates.
- Never edit checked-in generated fixtures manually. Regenerate them through
  the same Loom path consumers use.
- Generated example service methods fail closed with a transport-neutral Loom
  fault. They never create placeholder success bodies, files, or stream events.
  The design remains the source of declared transport responses. The service
  interface and the example stubs render one signature per method
  (`serviceMethodSignature` in `codegen/service/method_signature.go`); do not
  derive a stub signature separately.
- The example main of a server initializes only the services whose endpoints
  it passes to a transport handler of that server (`servedServices` in
  `codegen/example/example_server.go`). A hosted service without a transport
  gets no service, interceptor, or endpoint variables.
- HTTP transport generation defaults to all artifacts. API metadata
  `Meta("http:generate", "server")` omits client, client-type, client-path, and
  aggregate CLI files while retaining service and server packages. Stale client
  directories are removed only in the safe generator post-write phase.

## Authorization Ownership

- `codegen/service/authentication.go` emits one bound authentication factory for
  secured streaming methods. Endpoints, authorization checks and allocated
  aggregate authentication accessors share its ordered AND/OR policy.
  `security.RunCheck` owns original cancellation, replacement/nil contexts and
  error precedence for both admission and Protect; it carries no success cache.
  gRPC enables admission only for an initial envelope whose every security
  alternative is fully metadata-resolvable. `grpc.AdmitStream` retains the
  original metadata snapshot; `ServeStream` retains observation and method/service
  identity after callback context replacement. Full payload validation and
  ordinary endpoint security follow admission; mixed alternatives retain frame
  waiting and order. Keep `security/tla/stream_admission` and its runtime and
  generated regressions aligned.

- `expr.MethodExpr.selectedSecurity` selects ordinary and session declarations
  together before lowering. Validation, credential injection, finalized methods
  and transport mapping share that owner. Do not add transport-specific
  inheritance or separately inherit `SessionAuths`. Keep
  `expr/tla/authentication_scope` and its direct/generated regressions aligned.

- `expr` effective HTTP wire bindings own complete detached request, response and
  error mappings before validation. One plan retains physical slots, source field
  tuples, raw/Basic/Bearer codecs and security requirement membership. Coalesce the
  same source/codec; distinct sources may share a slot only as disjoint-framed,
  mutually exclusive auth alternatives. Derive credential body exclusions from
  selected bindings. Do not restore late auth/session/error mapping inference or
  independently walk API/service security scopes. Keep
  `expr/tla/http_wire_bindings` and generated HTTP/JSON-RPC/OpenAPI checks aligned.

- JSON-RPC WebSocket security admission consumes resolved wire bindings, including
  Basic and inferred session mappings. Bindings retain all source/tag roles when
  physical operations merge. Admission and finalization share the effective
  message-payload selector; every body credential role must survive with a valid
  string type. Reject unsupported handshake locations and displaced roles before
  generation; preserve supported per-message body authentication. Keep
  `expr/tla/jsonrpc_ws_admission` paired with mounted auth and DSL controls.

- Shared security analysis resolves credential selectors with `GoifyAtt` from
  the actual payload attribute. Basic username/password and API-key, JWT and
  OAuth credentials must use the same field identity as payload declarations;
  transport renderers consume that resolved selector. Scheme data also retains
  each credential's validated string type. `TaggedAttribute` resolves inherited
  tags through named payload layers before finalization. Cast named strings at callback and
  wire boundaries, and use NameScope to qualify conversions back to service
  types. `expr` rejects non-string, nullable and custom-Go credential types.
  Basic CLI arguments use physical strings; the shared field initializer owns
  conversion to service aliases. Their argument records retain each credential's
  effective default independently so flag metadata, field loading and physical
  pointer/value representation agree across HTTP and JSON-RPC. `cliPayloadArgs`
  projects direct inputs by logical AttributeData.Name before flags and payload
  builders consume them. Retain the first complete argument, whole-body carrier
  identity and ordering; never deduplicate allocated locals or change wire/server
  constructor arguments. Its fresh slice must not mutate cached argument storage.
  Keep `http/codegen/tla/cli_sources` and full generated CLI wire controls aligned.
  gRPC metadata
  records its physical decoded type
  separately from the service field type and uses `InitStructFields` for the
  constructor assignments, including pointer presence.

- `internal/securitygen.PlanBearer` resolves JWT/OAuth Authorization framing
  together with payload Required and authentication requirement membership.
  Only an optional credential omitted by another requirement may remain absent
  for a foreign Authorization scheme. Encoders frame once; decoders use the
  strict `security.DecodeBearer` before DSL validation and typed construction.
  `security.IsOtherAuthorizationScheme` owns foreign framing classification;
  malformed matching Bearer, singleton/AND-only credentials and global Required
  retain strict rejection. Authentication callbacks still select alternatives
  in declaration order and isolate failed contexts. Other mappings preserve raw
  values. Never normalize a constructed payload or guess prefixes from spaces.
  Metadata validators retain alias representation, nil guards and Unicode length.

- HTTP Basic header presence is analyzed once in `BasicAuthData`. Renderers
  send if either component is required or represented as present, and assign
  decoded fields only after a valid Basic header. An absent optional counterpart
  encodes as an empty component; it never suppresses its supplied counterpart.
  Missing/invalid headers require an error if either component is required;
  otherwise absent optional pointers remain nil. Basic analysis derives effective
  predicates from the actual credential occurrences against physical extracted
  strings. `RequestDecodePlan` owns the outer decoder's single error accumulator
  and validation barrier; Basic presence and supplied-value errors join ordinary
  errors before payload construction or authentication. Custom multipart keeps
  ordinary binding in its adapter. HTTP and JSON-RPC consume this same decoder.
  See the
  [presence model](../../../http/codegen/tla/basic_auth/README.md).

- `expr/authorization.go` owns strict coverage, typed bindings, and exhaustive
  enum/union classification. Authentication requirements remain independent.
- `codegen/service/authorization*.go` projects these expressions into typed
  evaluators, shared endpoint checks, and a deterministic static manifest.
- `security.Protect` owns execution and cancellation. Constructors and
  `Endpoints.Use` retain an outer check before short circuits and an inner check
  before service invocation. Runtime-owned middleware composition retains the
  original cancellation context and rejects continuations that drop the supplied
  context. Do not reduce this to optional interceptor wiring.
- Application checks must be read-only and repeatable. The framework guarantees
  the generated endpoint boundary, not raw service calls or externally replaced
  endpoint functions. MCP and agent adapters must explicitly enter that boundary.
- Keep policy engines, roles, relationship lookup, authorized query planning,
  per-message stream checks, and transaction-time authorization application-owned.
- Cover broken bindings and cases directly, compile generated code, exercise
  denial and middleware mutation through transports, and compare manifest bytes
  across isolated generation processes.

## OpenAPI Version Architecture

Loom emits OpenAPI 3.2.0 by default and supports a 3.1.1 compatibility target.
There is one DSL parser, one shared semantic IR, and one renderer.

- Gate emitted constructs by target version; do not fork parsers, IRs,
  generators, or canonical output paths.
- Render from the shared document model once. Apply 3.2-only transformations in
  `applyOpenAPI32`; for the 3.1 target, remove only incompatible members in
  `filterOpenAPI31` while preserving shared structures.
- Route version-dependent construction and document passes through
  `versionRouter`. Constructors declare bounded or open-ended `versionRange`
  values; when multiple constructors match, `construct` selects the one with
  the newest lower bound, while `mustConstruct` turns a missing required shape
  into an error. `runPasses` applies every matching pass in registration order,
  and the router carries constructor and pass diagnostics to the generated
  files. Keep non-versioned behavior outside the router.
- Keep shared JSON Schema features such as `contentSchema` when they are valid
  in both targets. Compatibility filtering must remove only version-specific
  members.
- Start OpenAPI contract changes in `http/codegen/openapi/internal/ir`; keep the
  `v3` package focused on rendering IR-owned decisions.
- `ir.decodeDiscriminatorMetadata` owns cross-field admission before schema
  mutation. Validate local fallbacks before representation allocation; include
  fallback references in identity, rewriting and pruning. Keep version projection
  in the existing router. See `ir/lean/discriminator_metadata` for the bounded model.
- `ir.projectPattern` owns the portable Go-to-ECMA pattern boundary. Project
  before schema reuse and rendering; carry unsupported projections through
  `Analyzer.Err` and `BodyTypes.Err` to the document generator. Runtime Go
  patterns stay unchanged. The bounded model and engine comparisons live in
  `ir/lean/pattern_projection` and `pattern_projection_test.go`.
- Prepare actual occurrence, codec and physical layout plans in
  `http/codegen/internal/representation` before sharing schema components.
  Built-in HTTP dispatch belongs to `internal/httpcodec`; SSE classification
  follows the emitted field type. Media labels alone do not establish codecs.
  `internal/byteschema` intersects alias bounds and projects the effective
  interval into checked base64 constraints. Complete representation schemas
  belong to the IR registry; renderers must not repair their constraints or
  identity. Schema-only preparation queries captured structure without resolving
  enum/default values or consuming example synthesis. Preserve component
  annotation ownership separately from representation-specific schema identity.
  Required defaults/enums retain checked declared values when the target codec
  is text, raw or custom, regardless of whether source resolution was builtin.
  Preserve the explicit Unsupported projection and known builtin sibling checks;
  retaining an annotation never proves custom codec output.
- Security component allocation belongs to the document IR. Identify a binding
  by authored scheme, kind, HTTP location, and credential name; canonicalize
  header case. Reserve authored names before allocating sorted binding variants,
  and project operation requirements through that registry. Keep API defaults,
  session-cookie names, AND/OR composition, OAuth scopes, and OpenAPI 3.2 external
  scheme URIs consistent. Do not change shared `SchemeExpr.Hash` for this purpose.
  Reject path credentials and OAuth2 mappings outside Authorization during
  OpenAPI generation; preserve their HTTP transport support and honor exclusions.
- Keep `http/codegen/openapi` limited to the active schema/document model and
  shared renderer helpers. Schema analysis state is per-render state owned by
  `internal/ir`; do not reintroduce the legacy JSON Hyper-Schema generator,
  package-global definitions, or compatibility aliases for those APIs.
- OpenAPI JSON uses Go 1.27 `encoding/json/v2` with deterministic ordering.
  Preserve two-space indentation, the final newline added by `codegen.File`,
  and configured prefix/indent behavior.
- Keep document identities at `internal/ir/document_identity.go`. Validate
  source operation claims before insertion into path maps, and final assembled
  operations after file-server and custom-method projection. Path templates
  ignore parameter names but retain static suffixes; exact method/path slots
  and nonempty operation IDs must be unique. For 3.2 media components, compare
  complete deterministic projected values before replacing inline values with
  references. Equal explicit claims may share; conflicting claims must fail.
- Project effective length bounds by the concrete wire value domain in the IR:
  strings use `minLength`/`maxLength`, arrays use `minItems`/`maxItems`, and maps
  use `minProperties`/`maxProperties`. Named and aliased maps retain their
  effective bounds. Bytes continue through the existing decoded-length and
  base64 representation owner rather than the generic string-length path.
- Primitive schema rendering and canonical fingerprints share
  `primitiveSchemaTypeFormat`; keep signedness and width decisions there.
  UInt32 uses `uint32`, and UInt64/native UInt use `uint64` under the existing
  native-width annotation convention. Preserve authored numeric bounds;
  format annotations do not replace runtime or JSON Schema assertions.
- All Loom-owned runtime, generator, test, fixture, and example JSON uses Go
  1.27 `encoding/json/v2` and `encoding/json/jsontext`. Generated code must use
  the same packages; do not emit or retain the legacy JSON package.
- OpenAPI output defaults to both canonical files. API metadata may select JSON
  or YAML alone; `codegen.File.RemovePaths` must remove only the stale sibling
  in the generator's deterministic post-write phase.
- Seed synthesized examples from stable schema and occurrence identity. Do not
  consume a shared traversal sequence that lets unrelated design changes
  perturb output. API-level example omission must retain explicitly authored
  examples in both JSON and YAML without mutating the evaluated design.
- Prepare builtin OpenAPI examples in `internal/representation` from retained
  `ValueContext` selection and synthesis results. Preserve ordered authored
  groups, metadata, explicit nulls, precision and selected union branches.
  Project those results through documentation plans; consumers must not sample
  again or infer a branch from the projected value. `DeclaredJSONValue` retains
  declared bytes and is distinct from target JSON output. The bounded model in
  `internal/ir/tla/nested_union_examples` checks selection ownership.
- Select untagged example branches against one shared normalized JSON value, while
  retaining the typed value for projection and final encoding. In particular,
  coerced `[]byte` fields must be matched as base64 strings without replacing
  the intermediate input or encoding it twice. Compare enum constraints using
  the same projected values emitted in the schema. Ambiguous wire shapes
  remain omitted. Never reinterpret typed bytes separately for a competing
  string branch; `internal/ir/tla/byte_examples` records that counterexample.
- The IR analyzer consumes prepared examples for bodies, nested parameters,
  response headers and async messages. Track the captured example position
  separately from structural schema authority: a neutral reusable schema can
  still own examples. Rebuilding a target plan invalidates its node-keyed
  attachments; reprepare them from retained source results without resampling.
  Documentation plans use JSON observation semantics while runtime plans keep
  their actual codecs. Preserve node-specific custom and SSE policies.
  `WithExampleValue` explicitly selects custom projection over original raw
  values; a nil projector uses retained declared values, including byte slices.
  The models in `expr/tla/schema_declaration` record these ownership boundaries.
- Treat stable schema names, canonical `operationId`, explicitly authored
  component names, and extension output as public framework contracts.
  Automatic request-body, parameter and header reuse compares complete contents,
  including retained examples. Response reuse is owned by `responseKeys` in the
  IR: compare status, explicit component name and complete shape, excluding only
  a present top-level description. Carry that description on `ResponseRef` and
  give the shared component neutral status text. Keep omitted-description policy
  and its 3.1 fallback together. Automatic names must not depend on error text;
  JSON/YAML codecs preserve absent versus empty description overrides. Different examples may prevent reuse
  and leave equivalent definitions inline; do not resample or discard examples
  to retain automatic references. Schema reuse remains separate. Compare
  downstream generated clients when changing this document structure.
- Canonical JSON member names for authored map keys come from
  `internal/jsonkey`. Expression examples and OpenAPI defaults share this
  conversion so integer widths, named scalar keys, and floating-point keys
  keep the same representation. Apply it recursively through default maps
  and arrays without changing byte-slice JSON encoding. Preserve custom JSON
  and text encoders, including pointer methods; the JSON library owns their
  representation and fallback behavior.
- An `Extensions map[string]any` field is tagged `json:"-" yaml:"-"`, so the type
  that declares it must also implement `MarshalJSON` and `MarshalYAML` through
  `openapi.MarshalJSON` and `openapi.MarshalYAML` with a `_Type` alias. A type
  without that pair accepts extensions and drops them at serialization.
- A scoped extension metadata key such as `openapi:<scope>:extension:<x-name>`
  needs a reader at the scope that emits it. Adding the writer alone leaves the
  metadata dead. Cover every scope in the semantic import round trip rather than
  in renderer unit tests alone.

The public feature inventory and metadata usage belong in
`docs/dsl-reference.md` and the consumer `loom` skill. Internal constructor,
filter, and serialization rules belong here.

## OpenAPI Importer Invariants

- Plan imported credentials once by physical wire slot, decoding codec and
  credential component. Payload attributes and HTTP mappings consume the same
  allocation; retain every scheme role and AND/OR requirement. Reject unequal
  per-scheme scopes within one conjunction rather than widening them. Partial
  operation projection retains root security only while a retained operation
  inherits it. Keep the credential model and actual import/auth tests aligned.

- Plan imported schema references once after component promotion and final error
  declaration allocation in `internal/openapiimport`. Retain declaration owner
  and source-path occurrence identity; resolve source component names separately
  from allocated Go names. Only eligible occurrences in a cyclic component become
  exact DSL-name lookups. Concrete constructor edges remain concrete, and residual
  initializer cycles receive a blocking subset diagnostic. Child renderers consume
  the same frozen plan. Keep the schema-reference models and direct/generated
  graph checks aligned with constructor behavior and metadata preservation.

- Use `encoding/json/jsontext.Pointer` for JSON Pointer token escaping,
  validation and decoding in both import and export paths. Keep the URI-fragment
  `#` and component-kind policy at the OpenAPI boundary. An invalid diagnostic
  owner path must retain its blocker rather than silently discard it during
  selection.
- Keep a field-level grammar ledger for each OpenAPI object the importer reads.
  Classify every parser field as preserved, conditional, lossy, rejected, or
  parser-only, and name the test or diagnostic that proves the classification.
- Compare that parser ledger with the fixed fields in the official OpenAPI 3.0,
  3.1, and 3.2 schemas. Add a raw-source guard when the standard defines a field
  that the parser does not expose.
- Guard version-specific fields even when the parser exposes their union for
  every version. Reject a 3.2-only field in a 3.0 or 3.1 document before render.
- Treat the OpenAPI 3.1 and 3.2 Schema Object as an open vocabulary at the
  parser boundary. Preserve JSON-compatible `x-*` extensions. Reject unknown
  non-`x-*` keywords unless Loom explicitly supports and tests them.
- Keep the canonical JSON and YAML fixtures in parity. Add every strict,
  cross-boundary construct to both fixtures.
- Test a newly supported schema construct in every legal component and
  operation location. Test shared response and naming policies as matrices,
  not only through the issue specimen that exposed the gap.
- Use the [bounded OpenAPI validation ledger](references/openapi-validation.md)
  to distinguish instance semantics, runtime behavior, compilation, structural
  linting and independent-process determinism.
- A strict import must render and evaluate. Forward generation must complete,
  and the generated Go code must compile. The generated OpenAPI contract must
  preserve the accepted source semantics.
- Keep an exporter-symmetry fixture for constructs that Loom emits and imports.
  Import the fixture, compile the service, regenerate OpenAPI, and compare the
  semantic contracts. Add each new inverse mapping to this fixture. If an
  exporter shape cannot round-trip, record and test the explicit rejection.
- CLI import round-trip comparison owns generated-reference normalization in
  test support. Preserve exact numeric tokens and literal values; rewrite only
  actual schema positions, then keep strict analysis as the admission gate.
  Resolve detached occurrences against a retained component registry, preserve
  local annotations and constraints, and prune only unreferenced generated
  targets after all occurrences resolve. Reject ambiguous overlaps instead of
  erasing differences. This adapter does not expand public importer grammar.
  Keep the comparison model and changed-generated-example controls aligned.
- Add each accepted response shape to an import-to-generation contract test.
  Schema-less errors use a concrete non-default error type and an explicit HTTP
  `Body(Empty)` mapping. This prevents synthesized default error headers. Never
  render `Error(name, Empty)`.
- Plan imported request bodies explicitly as raw, direct typed payloads, or
  selected-body envelopes. Schema-valued dynamic form maps use direct map
  payloads when body-only, selected `Body(...)` attributes when transport
  fields coexist, and raw contracts for multi-media or optional multipart
  bodies.
- The shared import planner admits cookie schemas only when they are primitive,
  resolving component schemas before checking. Array and object cookies fail
  with a blocking location-specific diagnostic before rendering.
- Parameter reference descriptions retain occurrence presence separately from
  the resolved component. Resolve by copying the parameter value and applying
  the override; never mutate the component or infer presence from nonempty text.
- Generated HTTP request code currently reserves the local name `body`. Until
  function-local names have an independent allocator, fail an imported
  request-body operation closed when a parameter or security field Goifies to
  `body`; never emit code known not to compile.
- Retain supported source values in the normalized import model before
  rendering. A diagnostic-only path cannot recover discarded values later.
- Preserve `byte` and `binary` string formats as separate OpenAPI contracts.
  Both formats use Loom `Bytes`. The rendered design must retain either explicit
  format in metadata instead of relying on an inferred Bytes schema format.
- Scope diagnostics by their JSON Pointer and owning layer, not by diagnostic
  code alone. The same code can identify a root omission or an operation
  blocker.
- Split safe document omissions before per-operation filtering. Root metadata
  must not make every operation fail during `--skip-unrenderable`.
- Apply component-closure filtering independently from document and operation
  filtering. Referenced component diagnostics must stay with their consumers.
- Copy a shared normalized schema before adding request or response metadata.
  Do not mutate a component schema while rendering one media type.
- Classify a schema as unconstrained only when it has no type and no retained
  assertions or applicators. Map that state to `Any`; do not infer `Any` from a
  missing type alone, because typeless constraints must continue to fail
  closed.
- Retain boolean schema values, authored type unions and the legacy `nullable`
  keyword until the shared import planner admits them. Reject false schemas,
  typeless properties and modern uses of `nullable`; do not infer an object
  assertion or apply OpenAPI 3.0 nullability in a modern dialect. Visit inline
  properties, array items, map values and union branches before rendering.
- Collapse a nullable composition only when its null branch is bare. Constraints
  and annotations on that branch must not disappear during normalization.
- Normalize `anyOf: [{}, {type: "null"}]` to unconstrained `Any`. The empty
  schema already accepts null, so the explicit null branch adds no contract
  semantics and must not force a nullable wrapper.
- Normalize `oneOf: [$ref, {type: "null"}]` to a nullable named alias only
  when the local referenced schema explicitly excludes null. Regeneration may
  use nullable `anyOf`; reject already-nullable targets because the two
  applicators are not equivalent in that case.
- Normalize JSON request and success-response `oneOf` named object/array branches
  to an explicit untagged union. Recursively admit primitives, named objects and
  arrays; reject maps, nested unions, inline object leaves and opaque codecs.
  Promote inline object branches to deterministic components; inline array
  branches remain unsupported. `expr.ValuePlan.JSONShape` lowers captured
  effective predicates. `internal/unionjson` projects service or HTTP member
  layouts and renders typed adapters; `pkg.MatchUntaggedJSON` owns orchestration.
  Independently require unique schema and actual Go matches with the same identity
  in both directions. Do not use numeric widths as schema discriminators, select
  the first successful branch, or mutate a destination before matching succeeds.
  Preserve API-level closed-object policy in the HTTP schema projection. See the
  [matching design](../../../internal/unionjson/README.md) and its bounded model.

- Normalize a scalar JSON Schema `const` without a sibling `enum` to a
  one-member Loom enum. Keep structured constants and combined `const` plus
  `enum` schemas rejected.
- Preserve canonical named aliases as OpenAPI components, including aliases of
  `Any`. Preserve the exact source component key even when multiple keys map to
  the same Go identifier. Canonical component identity takes precedence over
  ordinary primitive alias inlining.
- Preserve schema titles as first-class attribute annotations through import,
  attribute duplication, nullable wrapping, shared IR analysis, and rendering.
- Represent unconstrained payload and object fields with explicit presence so
  required JSON `null` is not mistaken for absence. An unconstrained error
  schema needs a concrete error envelope; only the envelope's root user type
  may receive generated error methods.
- Make every response-rendering branch honor `hasSchemaBlock`. Primitive and
  referenced-type shortcuts must not discard examples, defaults, or metadata.
- Preserve numeric-bound and compatible scalar-default siblings on a
  single-reference `allOf` as occurrence constraints over the canonical
  component. The importer emits `openapi:allOf:reference` metadata so
  regeneration retains the reference in `allOf`; other contract siblings
  remain unsupported until Loom can preserve them without flattening.
- Keep method-level error identities collision-safe across operations. Two
  responses with the same HTTP status must not share a generated error type
  when their schemas belong to different operation contracts.
- For a newly supported import construct, cover analysis, rendered source, DSL
  evaluation, and regenerated contract shape. Keep unsupported variants as
  source-specific diagnostics.
- Keep OpenAPI evolution policy outside Loom's module graph. Consumers own
  oasdiff versions, severity overrides, and ignores. Loom documentation can
  provide a pinned external workflow.

## Transport Invariants

- `RequestMetadataPolicy` owns forwarding trust. Verify the direct peer, then
  walk raw `X-Forwarded-For` header lines and comma entries right to left.
  Invalid or empty entries terminate with direct-peer fallback; never filter
  entries before selection or reinterpret headers in logging/security consumers.
  Keep `http/tla/forwarded_client` aligned with metadata, accessor and log tests.

- HTTP error response analysis retains the selected `Body` field in
  `ResponseData.ResultAttr`. Resolve its Go selector and client constructor
  pointer semantics through `transportFieldBinding`, just like other service
  field mappings. The shared response renderer consumes that selection when
  no body constructor is needed; explicit object bodies keep their constructor.
  `TestErrorBodyProjectionGenerated` covers exact wire bytes and typed client
  reconstruction, including raw JSON precision, renamed fields and pointers.

- Built-in HTTP errors have a public `ProblemResponse` representation distinct
  from `ServiceError` and from the selected transport body. The package-private
  `problemRepresentation` and `responseBodySource` owners select the source before
  generic body projection. Body, header and cookie fields read the same public
  document; client reconstruction uses shared transforms and field assignments
  before `ProblemErrorFromBody`, which consumes that complete document. The shared
  runtime document and `expr.ErrorResult` carry required Timeout/Temporary/Fault
  booleans, including false. Never infer them from status/code. Explicitly pruned
  objects omit unselected properties; scalar/empty bodies retain automatic
  headers. Keep the [property model](../../../http/tla/problem_properties/README.md)
  aligned with full/selected/mapped HTTP/JSON-RPC and unchanged gRPC controls.
  Keep complete, pruned, scalar and empty bodies
  on this path, and pass the whole error to custom formatters. The projection
  model and generated wire tests cover representation and member availability.

- `expr.HTTPEndpointExpr.validateResponses` owns the unique untagged application
  response requirement. Generators must not discard ambiguous authored responses;
  transport and OpenAPI analysis consume the same validated expression set.
  `transportir.buildResponse` stably places tagged alternatives before the
  default; renderers consume that order without reordering or swapping entries.
  The [response-order model](../../../http/codegen/internal/transportir/tla/response_order/README.md)
  checks priority preservation and selection; generated wire tests check Go.

- Length diagnostics receive only field identity, actual length, bound and
  comparison direction. `pkg.InvalidLengthError` owns their diagnostic contract;
  `codegen.renderLengthValidation` supplies these facts without passing rejected
  values. Keep classification, field metadata and the safe-message remedy intact.

- `pkg.MergeErrors` owns immutable accumulation of related validation errors.
  For two nonnil inputs it snapshots service fields and flattened original
  contributions, including Field and Remedy, while retaining original error
  causes. Nil is the identity. `History` snapshots on every read;
  `WithErrorHistory` is an explicit mutating setter that snapshots its inputs.
  Keep the ownership model in `pkg/tla/error_ownership` aligned with direct
  runtime tests and protobuf history round trips. Independent joined failures
  require aggregate transport classification, not validation-merge heuristics.

- Response-contract manifests classify unary HTTP, supported multipart, SSE,
  and plain HTTP WebSocket success cases explicitly. Multipart cases expose flat primitive and bytes
  parts from prepared transport data. Multipart SSE combinations remain an
  explicit limitation. Applications own codecs and fixtures.
  SSE cases own handshake, message, field-mapping, projection event type, and
  terminal-completion metadata. Declared pre-stream errors stay unary HTTP cases
  with the same stable case IDs.
- WebSocket cases own the `101` handshake headers, stream direction, inbound
  and outbound message types, and terminal behavior. Declared pre-upgrade
  errors stay unary HTTP cases with the same stable case IDs.
- gRPC response-contract cases own status codes, protobuf success messages,
  typed status details, and required header and trailer metadata. Unary and
  server-streaming cases are supported. A successful server stream terminates
  with clean EOF. Client-streaming and bidirectional completion contracts stay
  explicit generation limitations.
- gRPC CLI message examples consume the service's retained semantic example.
  Protobuf mapping owns field names, oneof allocation and message wrappers;
  it must not synthesize values or reselect union branches. Preserve omitted
  authored members. Diagnose and omit an unusable hint while keeping the
  command available; never advertise an unavailable example as JSON `null`.
- JSON-RPC response-contract cases own success result types, declared error
  codes, typed error-data names, and ID-less notification suppression. Unary
  and server-SSE cases are supported. Server-SSE requests with an ID retain a
  final response. ID-less streams suppress it. Other streaming completion
  shapes stay explicit generation limitations.
- JSON-RPC WebSocket params use the request-body declarations. Shared HTTP
  analysis must not also build or collect HTTP streaming-body declarations or
  initializers for these endpoints. Preserve the live params, result and error
  types, and ordinary HTTP WebSocket streaming bodies.
- Viewed JSON-RPC WebSocket and SSE method streams select a view with
  `SetView`, unless the design fixes it. Connection-level sends and WebSocket
  streaming-payload methods that return one result use the fixed or default
  view. When generation knows the view, emit only its body constructor;
  retain runtime selection for dynamic method streams.
- `service.StreamData` owns client operation names, contextual variants, result
  types and Close capability. Service interfaces, transport adapters and client
  interceptor wrappers consume those facts. Resolve server-only JSON-RPC methods
  at the role boundary; do not emit them on client streams. Effective SSE is
  retained in `MethodTransportData.IsSSE`. SSE adapters capture the endpoint
  context for Recv and accept the explicit context for RecvWithContext. Their
  direct HTTP constructor requires that context. Client wrappers preserve Close;
  shared SSE/gRPC interfaces retain gRPC CloseSend semantics. Keep the client
  stream contract model, typed-client fixture tests and ordinary transport
  controls aligned. Do not absorb gRPC cancellation or mixed-result accessor
  validation into method-set repairs.
- `codegen/service.clientInterception` owns client endpoint role and callback
  applicability before package declarations are allocated. Mixed primary and
  secondary bindings retain ordinary and message operations respectively;
  identical opening-only chains share declarations. Constructors, wrapper
  rendering and CLI command data consume its recorded names through
  `MethodData.ClientInterceptorWrapper`. Transport CLI analysis supplies its
  actual endpoint role, never a second applicability rule. Accessor records and
  canonical StreamData remain shared; do not duplicate them or add runtime cast
  fallbacks. Keep the client dispatch model, ordered callback matrix, mounted
  HTTP/RPC and CLI controls aligned. Shared server/client accessor coverage and
  JSON-RPC server wrapper shapes remain separate owners.
- `codegen/service.interceptorDeclarations` owns the shared Info/accessor method
  union and output-file placement. Build it from both side-local interceptor
  records during service analysis, preserving canonical method order and existing
  symbol identities. Rebuild it under file import alias analysis; never merge
  invocation lists or let a renderer discard client coverage by name. Keep the
  [accessor coverage model](../../../codegen/service/tla/interceptor_accessor_coverage/README.md)
  aligned with ordinary/streaming, imported-type and nonapplication controls.
  Field representation compatibility and JSON-RPC carrier shape remain separate
  contracts.
- `expr.MethodExpr.EffectiveStreamingResult` owns result-stream attribute
  identity, including ordinary/final-result fallback. Interceptor validation,
  field analysis and service stream analysis use it. Accessor references derive
  from canonical `ClientStream.RecvTypeRef`, including server accessors; do not
  substitute a JSON-RPC event interface or unary ResultRef. Keep the existing
  MethodData.StreamingResultRef mixed-result marker and eligibility rules.
  The streaming-result identity model and mounted HTTP/JSON-RPC clients cover
  this agreement; protocol-specific server wrappers and mixed unary endpoint
  dispatch remain separate owners.
- HTTP and JSON-RPC SSE handlers defer committing the stream until the first
  frame, except for the raw JSON-RPC `events/stream` GET listener, which opens
  eagerly so clients can observe readiness.
- Generated HTTP decoders check key membership for optional string query
  parameters. They use nil for an omitted key. They use a nonnil pointer for
  an empty or nonempty value. Generated clients emit the key for every nonnil
  pointer.
- Raw JSON (`Any`) object properties use nil for absence, including named
  types. Their transforms apply defaults only to absent optional properties
  when the target context uses defaults and is not a pointer context. Explicit
  JSON null and concrete zero values retain their bytes. Custom Go type
  overrides keep their own representation rules.
- An optional payload attribute selected with `Body`
  (`RequestData.OptionalBodyAttribute`) whose service field can be nil or
  absent, which is any attribute but a primitive with a default value, is
  sent only when it is not nil or absent: HTTP clients send no body and JSON-RPC clients omit
  params. Servers decode an empty body, empty form, or absent JSON-RPC params
  to a nil or absent attribute. A union gets this from its empty
  discriminator. A nullable attribute (`OptionalBodyNullable`) is checked with
  `Present()` and an `Any` attribute against nil. Explicit presence bodies are
  declared with the type that the payload constructor takes, including the
  `loom.Nullable` collection elements of JSON decoding, and are passed to it
  by value; the payload field holds the value without a pointer. Their
  validation requires presence only when the body is required
  (`requestBodyRequired`). JSON-RPC servers decode absent params as `{}`
  only when the body is neither optional nor explicit presence
  (`RequestData.ExplicitPresenceBody`), so absent params of a required
  explicit presence body, or of a nullable or `Any` payload, fail with
  `missing_payload`. A nullable object body gets its own transport
  struct; its `Validate` function takes the `loom.Nullable` of that struct,
  and `TypeData.ValueRef` declares the decoded variable. HTTP, WebSocket and
  JSON-RPC response decoders consume this same value reference; `VarName` is
  the declaration name and cannot substitute for a nullable body value. The
  shared JSON-RPC response writer applies this to success and error bodies.
  A non-nullable object
  (`OptionalObjectBody`) is decoded into a pointer that is set to nil for an
  empty body, is validated only when present, and is passed to the payload
  constructor, which leaves the attribute nil for a nil body. A primitive
  pointer (`OptionalPrimitiveBody`) is decoded the same way into `new(T)`,
  and the constructors read it through `*body`. An array, map, or bytes body
  stays nil for an empty body and is validated only when it is not nil
  (`payloadBuilder.optionalBodyValidateRef`); the constructor of every such
  nil-able body (`InitData.ReturnIsOptionalBody`) leaves the attribute nil
  for a nil body. The client CLI body flag of such an attribute is optional;
  for these nil-able bodies,
  `cli.PayloadInitData.ReturnTypeAttributeFlag` sets the attribute only when
  the flag is not empty, and the CLI declares a primitive body as a pointer
  unless it decodes a user type, such as an alias, as JSON
  (`isCLIBodyPointer`). The client body builder of an optional alias of a
  primitive dereferences the field, which the encoder checks for nil. The
  server validates a request body that is an alias of a primitive, an array
  or a map by value: its `Validate` function reads the value without a nil
  check (`buildUserRequestBodyTypeDetails`), and the decoder passes it the
  body, not its address (`requestBodyValidateRef`). Flags actually decoded as
  JSON consume the retained `Source.Example` through the client body
  `TypeData.Value` runtime plan. Preserve the projected JSON bytes through
  formatting, including explicit null and numeric precision. Do not reselect or
  synthesize examples at the CLI boundary. Plain byte/text flags keep their
  location codec even when the eventual HTTP body uses JSON.
- An empty serialized CLI example means no hint; JSON `null` remains a value.
  Omit unavailable hints from both JSON error constructors and help. Leave an
  optional flag out of a sample invocation when its example is unavailable;
  omit the invocation when a required flag lacks a usable example. Individual,
  aggregate and top-level help must not print empty Example sections. Keep the
  command and parser available. Missing/mismatched retained plans are generation
  errors, not an example-omission policy.
- The HTTP and JSON-RPC client CLI command parsers never name a service type,
  because their package does not import the service packages. An object
  payload without flags, such as an object type without attributes, gets a
  payload builder without parameters in the client package (`buildFlags` in
  `http/codegen/client_cli.go`).
- JSON CLI flags serialize defaults deterministically, including nested
  collections. Optional collection attributes selected with `Body` propagate
  their defaults to the CLI flag metadata. Boolean map keys use the shared
  `internal/jsonkey` JSON options; generated HTTP and JSON-RPC CLI builders
  select `http/cli.UnmarshalJSON` when their types contain boolean map keys.
  Custom JSON/text codecs retain precedence, and the runtime import receives
  a collision-free alias in each payload-builder file.
- `loom.JSONOptions()` exposes the shared boolean map-key options to generated
  code. Apply them at typed branch decoding and value encoding in both service
  and HTTP union codecs, in `Optional` and `Nullable`, and at HTTP body codec
  boundaries. Outer JSON options do not propagate through legacy custom
  `MarshalJSON`/`UnmarshalJSON` methods that start new JSON operations. Preserve
  explicit `json.Deterministic(true)` in custom marshalers. The model under
  `pkg/tla/json_options` reproduces the outer-only failure and the incomplete
  union-only repair, then checks nested option handling at all three layers.
- Ordinary unary HTTP handlers delegate request context, observation, decode,
  invocation, response encode, and failure routing to the typed runtime helper.
  A response encoder failure that occurs before commit is encoded through the
  generated error contract; a post-commit failure goes only to the application
  failure handler.
  Other generated HTTP handlers use `HandlerLifecycle` for context,
  observation, and commit-aware failure routing. File and raw-body branches
  also delegate writes and cleanup. Generated code retains typed result and
  stream adapters.
- `expr.validateHTTPPathWildcards` validates the complete inherited route for
  endpoints and Files before generation. Parameters must start a segment;
  static suffixes are allowed, composite or embedded parameters are rejected,
  and catch-alls must terminate the path. Keep this grammar aligned with
  `ExtractHTTPWildcards`, path construction, and mux matching.
- Generated HTTP path builders return escaped paths (`renderPathInitCode` in
  `http/codegen/paths.go`). They escape route literals at generation time and
  string-like values at run time with `http.EscapePathSegment`, or
  `http.EscapePathRemainder` for a catch-all value. Each value is formatted by
  the type of its own param. Clients build the request URL with
  `http.RequestURL`, which keeps the escaping in `URL.RawPath`; a route whose
  builder returns an unescaped literal keeps `url.URL.Path`. The muxer matches
  escaped route literals against the escaped request path and preserves the
  authored pattern in request metadata. Honor a parent router's remaining path
  without modifying the request URL or escaping that remainder twice.
- `Muxer.RawVars` is a required raw-capture contract; `Vars` retains decoded
  values. Generated array path decoders read raw captures and delegate to
  `http.DecodePathArray`, which splits literal commas before unescaping each
  element once. Normal and multipart decoders share this runtime operation.
  Never reconstruct raw captures from decoded values or silently fall back to
  decoded splitting for custom muxes. See the
  [path decoding model](../../../http/tla/path_decoding/README.md).
- Generated wildcard static-file routes delegate target resolution to
  `http.NewStaticFileServer`. Directory targets map the captured request suffix
  below the target, while file targets serve the same file for every matching
  route. Generators must not infer target kind from a path extension.
- HTTP SSE `id`, `event`, and `retry` mappings follow the service type pointer
  semantics recorded in `SSEData` (`IDPointer`, `EventPointer`,
  `RetryPointer`, `IDDefault`, `EventDefault`). Servers nil-check and
  dereference optional fields; clients leave them nil when the frame omits the
  field and assign defaults to defaulted ones. JSON-RPC SSE does not map these
  fields onto SSE frames.
- An HTTP SSE data mapping of an optional primitive without a default
  (`SSEData.DataPointer`) follows the same pointer semantics. A String field
  is raw text: servers write the dereferenced value, or no data for nil
  (`writeSSEDataFieldPayload`), and clients leave the field nil for empty
  data. Other optional primitives stay JSON literals, `null` for nil, and
  clients decode them with the decoder (`renderSSEParseAssignment`).
- Protocol errors must use event types compatible with the relevant client
  contract.
- Generated HTTP variables named after path params, query params, headers,
  and cookies come from one variable scope per request, path builder, and
  response or error response (`http/codegen/transport_var_scope.go`). Each
  scope reserves the identifiers that its generated functions declare or
  import. It also allocates the locals derived from each variable, such as
  the raw string form of a header, into `AttributeData.Locals`. Templates use
  those fields and never append a suffix to `VarName`. When a template gains
  a local in one of those functions, add it to the matching list.
- Every per-service HTTP, gRPC, and JSON-RPC file that can reference service
  types imports the `struct:pkg:path` packages itself, as do the service,
  endpoint, client, views, interceptor, and authorization files. Transport
  files append `Service.UserTypeImports`, which `service.SetUserTypeImports`
  sets before any transport data is built. `codegen.File` prunes the unused
  imports. The generators add only `struct:field:type` imports afterward. Do
  not reintroduce a blanket pass that hides a missing import in a direct file
  builder.
- Declarations of types placed with `struct:pkg:path` use their canonical Go
  name, just like qualified references. `NameScope.goFullValueTypeName` keeps
  the existing conservative reservations that protect generated helper names,
  then returns the canonical name for a relocated declaration. Removing those
  reservations lets a union discriminator steal an authored `ChoiceKind`
  name. The models under `codegen/tla/external_type_names` check declaration
  consistency and preservation of authored names; the shared transport fixture
  covers all six type shapes and an authored union-discriminator name.
- A file imports a `struct:pkg:path` package under an alias, such as
  `security2`, only when its name clashes with another import that the same
  file lists (`codegen.AliasClashingImports`). A file builder lists its imports
  first, then renders its sections with the data that `FileData` returns for
  them (`service.Data.FileImports` and `ServicesData.WithPackageNames`). The
  name scopes of that data qualify the renamed packages with their aliases
  (`codegen.NameScope.PackageName`). Resolve every relocated type qualifier
  through the scope of the data, never with `Location.PackageName`, which
  only names import specs and the packages themselves. Derive identifiers that
  other files use, such as transform helper, constructor, and method type
  names, from the package names (`NameScope.WithoutPackageNames`,
  `AttributeContext.NamePkg`). Code that
  adds imports to a header after the builder, such as the JSON-RPC encoders,
  passes them to the builder instead. Import pruning keeps or removes imports
  of one name together, so a file that lists two imports of one name compiles
  only when it uses neither, and the aliases change only code that did not
  compile. Do not replace this with a blanket rename or an import fixup after
  rendering. Compiled pattern validations import `regexp` under an
  alias when the file already has another import named `regexp`.
- `service.PackageName` names a service package and seeds the HTTP service
  import aliases. It adds the `svc` suffix when the service has a transport and a
  `struct:pkg:path` package of its types has the same name, because the
  transport files import both. A transport-less service keeps its name. `SetUserTypeImports` rejects a type placed in the package of the
  service that uses it. Streaming payload and result types keep their own
  locations (`StreamingPayloadLoc`, `StreamingResultLoc`); do not derive them
  from the unary payload or result location. HTTP request, response and
  WebSocket streaming bodies (`httpStreamingBody`) strip `struct:pkg:path`
  from every user type they refer to, including the elements of a collection
  body, so the transport declares its own types and converts them to the
  relocated service types.
- HTTP and JSON-RPC transport and example files import a service package
  under a `svc` alias, such as `ssvc`, when its name equals an import of
  those files (`transportGeneratedImportNames`) or a name that they declare
  where the package would not compile under its own name
  (`transportGeneratedLocalNames`, such as the server receiver `s` or the
  request `r`). `newServiceImportAliases` allocates the aliases.
  `testingx.ServicePackageShadowNames` collects those names from
  generated code by scope, and the HTTP and JSON-RPC local name tests fail
  when the list misses one and compile services named after each. Add a new
  generated name to the list. The example server main allocates its service
  imports in its own scope, which reserves its locals
  (`serverMainLocalNames` in `codegen/example`).
- gRPC analysis copies neutral service data and assigns transport import aliases
  from `newImportAliases`. `newTransportNameScope` reserves the single fixed
  import/local inventories before both protobuf/service import allocation and
  metadata-local allocation. Keep receiver `c`, locals such as `md`/`ctx`, and
  protobuf-shadowing `reqpb` in that shared owner. Generated import coverage and
  `TestGRPCServiceLocalSymbolsScopeInventory` check actual rendered scopes.
  `ServiceData.PkgName` is the protobuf import alias; `ProtoPkg` preserves its
  declaration and wire identity. Neutral service names remain unchanged.
  `metadataVarScope` then reserves actual service/view/protobuf/custom imports;
  request locals and shared response header/trailer locals avoid these bindings,
  including file-specific `struct:pkg:path` aliases from `fileData` reanalysis.
  Do not mutate neutral nominal scopes or add per-renderer escape policies.
  gRPC example files retain service/server/protobuf bindings in private
  `exampleServerServiceData`. Header allocation and every parameter, constructor
  and registration reference consume those exact bindings, without modifying
  cached transport data. Preserve allocation order, service subsets and ordinary
  output. The model under `grpc/codegen/tla/import_aliases` covers reciprocal
  reservation, suffix ordering and retained example bindings; generated compile,
  metadata round-trip, custom import and host-isolation tests cover the seams.
- `conversionCanFail` in `grpc/codegen` owns each source/target conversion's
  transitive error capability. Root constructors, shared helper declarations and
  helper calls query their own conversion pair and physical source context.
  `elementNeedsAdmission` owns nil-message collection admission before wrapper
  normalization; `transformElement` emits it before inline/helper conversion.
  Target wrappers alone do not make outgoing native collections fallible. Keep
  nil optional fields separate, and retain source variables for empty objects.
  Type files and CLI builders must import the error constructor they emit.
  `objectHelperTransformAttrs`
  detaches the body context before selecting helper-local fallibility; never
  inherit an unrelated root's error flag or encode that flag in helper names.
  The model under `grpc/codegen/tla/helper_fallibility` and the mixed-order
  helper regressions cover recursive calls, pure children and propagated Any
  errors. Keep collection-occurrence admission separate from optional fields.
- `makeProtoBufMessageR` lowers scalar aliases to their native protobuf types
  while retaining effective constraints and defaults. Root scalar wrappers must
  traverse this same lowering before validation or conversion. The protobuf scope,
  renderer and `protoBufTypeContext` own physical presence: all scalar message
  fields use explicit presence independent of requiredness or defaults.
  Validation must see that layout before conversion. Scalar wrappers convert
  pointer fields to/from service values; collection elements and scalar oneof
  branches remain values. Defaults apply only to absent optional fields.
  Required fields with defaults still require presence, and service encoding
  must preserve explicit zero values. See `grpc/codegen/tla/scalar_presence`.
- gRPC streaming request finalization must merge the validation of the
  unwrapped payload attribute into the prepared message validation. The
  prepared validation is already non-nil, so `initAttrFromDesign` alone drops
  root array, map and scalar constraints. Preserve element constraints and
  object requiredness, and test both ordinary streams and stream envelopes.
- The fields, oneofs and oneof fields of a gRPC message share one namespace.
  `newProtoMessageNames` (`grpc/codegen/protobuf_message_names.go`) allocates
  their names per message, and the proto renderer, `checkMessageFields`, the
  Go transforms and the validation code (through the `messageFieldScope`
  interface in `codegen`) all use it. Pass the containing attribute: all scalar fields add synthetic oneofs
  to protoc's Go-name reservations. `FieldNames` supplies field selectors, branch selectors, and
  separate wrapper type suffixes. `protoGoName` is only the unallocated base;
  never derive a message selector from it or from `Scope.Field` alone.
  Go allocation follows descriptor order and reserves generated methods and
  getters. Map-entry types can rename a branch wrapper without renaming its
  field. If protoc itself would emit conflicting selectors, oneofs take extra
  `_oneof` suffixes and a `ProtoReflect` field takes `_field` suffixes. Preserve
  existing wire names when protoc's Go selectors are valid. The bounded model
  in `grpc/codegen/tla/field_names` checks allocation order and repair; direct
  tests compare against protoc descriptors and the pinned Go plugin.
- `protoBufScope.Field` ignores service `struct:field:name` metadata without
  mutating it. Protobuf validation must use protoc field names even before
  transport normalization removes service metadata; service structs keep their
  authored field overrides.
- The gRPC client CLI decodes the request message flag with `protojson`
  (`InitArgData.ProtoMessage` sets `cli.FlagData.Unmarshal`), because
  `encoding/json/v2` leaves oneof fields nil. `protoJSONExample`
  (`grpc/codegen/client_cli_example.go`) builds the flag example in the same
  protocol buffer JSON mapping from the proto message attribute and
  `newProtoMessageNames`. A message without a union keeps the example that the
  attribute generates, with keys renamed to the protocol buffer names, so its
  example and the examples generated after it do not change.
- The part of an object key before a colon is the attribute name, and the
  part after it, as in `"n:m"`, is the name of the transport element
  (`expr.AttributeName`, `MappedAttributeExpr.ElemName`). A mapped attribute
  names its attributes and its required attributes without the suffix, and
  `MappedAttributeExpr.Attribute` restores the suffix on both. Go fields are
  named after the attribute, so the shared and gRPC transforms walk objects
  through the mapped attribute and never call `ElemName`, and validation
  looks attributes up in the mapped attribute by attribute name. Service
  JSON tags, union branch kinds (`expr.UnionVariantTag`), the names of
  `OneOf` block unions and of their promoted branch types, view attribute
  lists and validation error names use the attribute name. HTTP and
  JSON-RPC bodies use the suffix as the JSON, form and XML name of the
  field, and so do the OpenAPI schemas and examples, shared runtime projection
  of JSON CLI body examples, and the JSON fields of the HTTP untagged union
  branches: they name the field with `expr.JSONFieldName(expr.ElementName(key),
  att)`. gRPC ignores it: the protocol buffer field and the Go fields of the
  service and pb types use the attribute name. The object validation checks
  the JSON names of the service fields, named after the attribute, and
  `HTTPEndpointExpr.validateBodyElementNames` checks the element names in the
  objects of the HTTP and JSON-RPC bodies of each endpoint, without the
  attributes mapped to params, headers and cookies, so that a suffix that
  collides only in a gRPC type or outside a body is accepted. An attribute
  of an explicit `Body` function or of a type with a reference inherits the
  referenced attribute with the same attribute name
  (`AttributeExpr.FindAttribute`, `objectAttribute` in `Inherit`), and the
  inherited required names use the body keys, so `Attribute("name:n")`
  behaves as `Attribute("name")` except for the element name;
  `validateBodyRequiredPayloadAttributes` and the response body checks
  compare attribute names, and `validateBodyRequiredKeys` rejects a body
  `Required` name that omits the suffix of its key. Header, Param, Cookie,
  MapParams, Body and route wildcard names select payload, result and error
  attributes by attribute name the same way: the expr validations and
  `initAttr`, `initResponseCookies` and the response body finalization, the
  HTTP transport IR and the payload, response and error builders look the
  attribute up with `FindAttribute` or `objectAttribute` and test
  requiredness, defaults and pointers with the key it returns
  (`transportir.Request.BodyOriginKey` for an attribute selected with
  `Body(name)`, `originKey` in the HTTP code generator). The element name of the
  mapping, not the payload suffix, names the header, parameter or cookie. A route wildcard names the
  element of a param, so `Param("key:k")` is the path parameter of `{k}`
  (`HTTPEndpointExpr.PathParams`, `QueryParams`, `ensureRouteParams` and
  `RouteExpr.Validate` match wildcards by element name, and the path builders
  look path parameters up by `HTTPName`); a wildcard that names the attribute
  of a mapped param is rejected.
- Protocol buffer messages always live in the pb package of the service.
  `makeProtoBufMessage` strips `struct:pkg:path` from the message attribute as
  well as from its user types. `makeProtoBufMessageR` resolves explicit message
  names onto normalized occurrences, including nested references; wrapper
  construction retains the name across physical lowering. Declarations,
  references, converters and response contracts consume that normalized name.
  `expr.validateStructMeta` rejects missing or multiple names before lowering;
  no consumer may choose first or last among competing names. Go references
  apply `protoGoName` to the single name. Never add renderer-specific name
  recovery or a fallback for conflicting explicit names.
- The messages that wrap a named primitive, union, array or map (`wrapAttr`,
  `wrapUnionBranch`, `wrapCollectionUserType`) are new user types with their
  own identifiers. Copies and examples share user types by identifier, so a
  wrapper that kept the identifier of the type would replace the type
  wherever else it appears, such as a recursive branch of a union or a field
  of the same named primitive.
- The message that wraps a named array or map, and the message that wraps an
  array or map nested in an array or map, such as `ArrayOfHolder`, are
  collection messages (`isCollectionMessage`) that the gRPC transforms convert
  inline. Recursion detection (`isInlineRecursive`) skips them and finds the
  object type that reaches itself through them, which converts through a
  helper. The loop variable of a map transform (`transformMapElemVar`) never
  shadows the variable of the target map, because the map depth that names
  it does not tell nested maps apart when a message is recursive.
- `wrapperAttribute` derives contents requiredness from the wrapped type.
  Repeated/map contents may decode to nil inside a present message and still
  represent an empty collection; scalar/union contents retain presence checks.
  Preserve authored length predicates and required containing-message presence.
  Root, named, nested and endpoint-shape paths consume the same constructor,
  including anonymous contract allocation. Do not restore caller-selected
  requiredness or add transport validation exceptions. See the
  [collection contents model](../../../grpc/codegen/tla/collection_contents/README.md).
- Named map and array union branches use their collection message wrappers.
  `unionBranchMessageHelper` must collect conversion helpers for both kinds,
  as well as nested unions. Keep a wrapper for every selected branch, including
  nil and empty maps. `grpc/codegen/tla/map_union_branches` checks this presence
  rule; the map-alias fixture tests it through protobuf serialization.
- Protocol buffer messages are identified by name, and the name scope hashes a
  user type by its name, so a generated message and a design type with the
  same name become one message. The top-level messages generated for a
  method (`<Method>Request`, `<Method>Response`, the streaming request, stream
  item and envelope, and `<Method><Error>Error`) take their names from
  `endpointMessageName` (`grpc/codegen/protobuf_endpoint_names.go`), which
  numbers the name when a design type that a message of the service declares
  has it and a different message (`designMessageShapes`). A design type with
  the same message still shares it, so designs without a conflict keep their
  output.
- Nested collection wrappers reserve names in the same registry as endpoint
  and anonymous messages. `collectionMessageName` shares a declaration only when
  its wire shape and detached complete effective contents contract match.
  Preserve ordered predicates, defaults, nullability and storage facts; named
  edges retain their declaration ID and occurrence facts without traversing their
  mutable children. Never merge predicates from distinct collection occurrences.
  The [wrapper contract model](../../../grpc/codegen/tla/wrapper_contracts/README.md)
  checks faithful predicates, helper closure and equal-contract sharing.
  `collectMessages` keys emitted declarations by generated Go name and checks
  the protobuf name and field shape before reuse. Shape comparison retains each referenced
  message name, even on repeated or recursive edges. This catches explicit-name conflicts at
  every depth and rejects distinct protobuf names that map to one Go type.
  Recursive normalization rebinds repeated canonical type IDs to the first
  message, so copied inline-object graphs cannot retain unnormalized aliases.
- A `struct:name:proto` name can give one message to several service types,
  such as customized copies of a type. `registerProtoMessage` keys messages by
  that name and rejects two uses with different fields. `nameProtoConverter`
  gives each service type its own converter to the message in each of the
  server and client packages. The first converter keeps the message name, so
  designs without shared messages keep their output.
- The gRPC client tells the errors of a method apart only by the type of the
  status detail message, so its type switches have one case per message
  (`clientErrorCases`). Errors of one type that share a `struct:name:proto`
  message share the first converter; errors of different types that map to
  one message fail generation.
- protoc resolves the messages of an rpc in the scope of the service first,
  where the rpcs are defined. The service definition qualifies a message that
  has the name of an rpc with the package (`rpcMessageRef`), such as the
  named array `Tags` of the method `tags`.
- An empty object payload or result is an object, not `Empty`: the gRPC code
  generator tests the payload and result against `expr.Empty` and uses
  `isEmpty` only for messages without fields. The converter of such a message
  takes no argument, so the codecs discard the value that they assert
  (`valueVar`), while the stream converters always take the stream item.
- `http.WebSocketStream` owns attachment, one physical reader, bounded inbound
  admission and terminal publication. Generated endpoints supply typed adapters,
  not independent watcher/done/closeOnce state. `WebSocketConfig` owns byte/count
  limits and the hook; `SetConn(ctx, conn, cancel)` configures before adoption and
  binds physical lifetime there. Hook replacements own retiring their inputs;
  nil results retain terminal attachment state after an HTTP upgrade.
  Queue and current-decode reservations share the byte/count budgets; reserve
  space for one in-flight maximum message and count empty messages. Release a
  decode reservation only after its callback returns. JSON-RPC uses that seam
  for envelope parsing and retains its own logical dispatch/demultiplexing.
  The first terminal transition publishes cause and cancellation and releases
  the socket. Natural read failure retains admitted FIFO work for live-context
  readers; overload and explicit close discard it. Preserve raw Close's
  immediate-release contract. Shared handler cleanup, status close and final
  JSON close bound terminal writes independently of application writes, always
  release the socket, and preserve the original handler error/panic. Final JSON
  close shares one second across writer acquisition, final write and best-effort
  peer termination. Its sole pump discards bounded late frames while cleanup waits;
  local closed state must not masquerade as peer termination. Deadline expiration
  after a successful write adds no cleanup error or delivery guarantee. Keep
  terminal framing classification in `releaseConnection`: a raw native
  `ErrCloseSent` means close framing already completed and skips peer waiting.
  Preserve wrapped application encoding errors and independent physical-close
  failures; ordinary data writes must still fail after close framing. Keep
  CloseAndRecv's null half-close nonterminal. Align both message-admission and
  lifecycle models with real-socket and generated transport regressions.
- Keep JSON-RPC envelope validation, batch framing, notification suppression,
  HTTP/SSE negotiation, WebSocket setup, and final stream-response decisions
  in `jsonrpc`. Generated JSON-RPC code supplies typed dispatch, error, and
  stream adapters only.
  `jsonrpc.ReportFailure` owns optional diagnostic callback invocation for
  runtime and generated adapters. Preserve the original context, writer and
  wrapped error, and invoke provided callbacks synchronously once. Never call
  optional errhandler directly or normalize it only in server.New: exported
  per-method handler constructors must share the same policy. Keep protocol
  suppression and failure observations at their existing call sites; this owner
  neither writes responses nor recovers application callback panics.
  WebSocket typed dispatch uses one `writeWebSocketRequestFailure` renderer
  for decode and endpoint failures: original HasID selects suppression or the
  typed error sender, and only sender failure escapes to the receive loop.
  Preserve originating invalid-params observation; request rejection is not a
  service-stream close event. Never suppress all notification dispatch errors in
  ReceiveWebSocketRequest, where they can represent native I/O failures.
- `jsonrpc.Response` owns the wire form of every JSON-RPC response
  (`Response.MarshalJSONTo`): a success response always has a `result`
  member, null for a method without a result, and an error response has none.
  Build responses with `MakeSuccessResponse` and `MakeErrorResponse`; do not
  add an envelope type with an `omitempty` result, because JSON v2 omits
  `""`, `[]` and `{}` as well as null. `Request.Params` is likewise
  `omitzero`: a request omits `params` only when `Params` is nil, so HTTP and
  WebSocket clients and notifications send empty strings, arrays, objects
  and null alike. Generated HTTP clients leave `Params` nil only for a method
  without payload and for an absent `OptionalBodyAttribute`, so a typed nil
  never reaches the envelope as null.
- The unary endpoint of a generated JSON-RPC HTTP client whose payload has an
  ID attribute checks the response to a notification, which the encoder sends
  for an empty ID, with `jsonrpc.DecodeNotificationResponse` and returns the
  zero result instead of decoding a response
  (`writeJSONRPCNotificationResponse` in `jsonrpc/codegen/client_sections.go`).
  Its locals are names that service imports already avoid
  (`transportGeneratedLocalNames`).
- Generated JSON-RPC WebSocket clients share one `jsonrpc.WebSocketClientConn`
  per connection. It is the only reader: it assigns connection-wide request
  ids, routes each response by id, and fails every waiter when the read fails.
  Registration checks, under the same lock, that the connection is still
  usable, so no request registers after that fan-out. Streams hold a
  reference; the last release or the client closes the connection. The last
  release marks the connection closed in the same critical section that
  drops the reference, so a concurrent `Acquire` cannot take a connection
  that is being closed. Generated
  streams wrap `jsonrpc.WebSocketClientStream` and only decode results. Check
  changes against `jsonrpc/tla/WebSocketClientDemux.tla`.
  Explicit closure errors share `jsonrpc.ErrStreamClosed`; retain the more
  specific stream and connection sentinels. Empty-queue receives still check
  the connection error. In-flight socket-closed writes retain both their
  semantic cause and socket error, while cancellation and unrelated failures
  remain distinct. Keep `jsonrpc/tla/CloseErrors.tla` paired with direct socket
  races and compiled client tests, including server-streaming receives.
- JSON-RPC reuses HTTP file builders for its encoders, decoders, types and
  paths. `updateHeader` (`jsonrpc/codegen/header.go`) moves only the imports
  below `genpkg/http/` to `genpkg/jsonrpc/`, matching whole path segments;
  service packages such as `genpkg/http_` and module paths that contain
  `gen/http` keep their import paths.
- JSON-RPC mapped error emission shares `mappedErrorBody` across unary, SSE and
  WebSocket renderers. Consume the prepared response EncodePlan, including its
  public ProblemResponse source, and normalize the original error to retain
  outer safe-message/remedy ownership. SSE endpoint-returned service failures
  delegate to the existing terminal SendError mapping; protocol failures keep
  their generic data and request-ID suppression. Do not duplicate that mapping
  in the endpoint handler or change the terminal state algorithm.
- Preserve the generated public `ServeHTTP` middleware and policy chain when
  adding JSON-RPC dispatch branches.
- Generated gRPC clients decode remote errors inside the `RemoteFunc` callback
  passed to `grpc.NewInvoker`. Keep request encoding and response validation
  outside that mapping so their errors retain their identity and fields.
  Decode generic error details with `grpc.NewServiceErrorWithCause` in unary
  and stream clients so the original RPC error remains inspectable. Received
  fields and history stay authoritative; preserve the first-decodable-detail
  rule and existing custom-error mappings.
- The gRPC remote-method builder retains a stream after initial-payload send
  EOF (including wrapped EOF). Final status belongs to `Recv` or `CloseAndRecv`;
  non-EOF send failures still terminate opening. Keep this in the common
  stream-envelope branch, with the `grpc/codegen/tla/initial_send` model and
  compiled client/bidirectional regressions aligned.
- Keep gRPC request context, metadata application, status conversion,
  observation, and clean stream completion in `grpc.ServeUnary`,
  `grpc.ServeStream`, and `grpc.EncodeServerError`. Generated gRPC code supplies
  protocol buffer adapters and typed `grpc.ErrorMapper` callbacks. Do not copy
  the status lifecycle into generated service methods.
- `grpc/stream_context.go` owns per-message cancellation and native I/O lifetime.
  `ServeStream` retains synchronous Decode/Handle and original request cancellation;
  `NewStreamInvoker` owns the client opening context through response decoding.
  Generated adapters retain that context and delegate native I/O to the owner.
  Keep one pending native send and receive; final CloseAndRecv reserves both.
  Context cancellation latches a terminal cause and returns without joining native
  server I/O, which needs handler return to finish. Detach outgoing protobufs,
  keep receive results private, and stop/join completed operation callbacks.
  Admission is explicit: rejected operations cannot finalize a healthy client.
  Every terminal transition detaches parent watches, including registration races.
  CloseSend remains a half-close; final receive and opening/decoding failure release
  the client lifetime. Operation kind is separate from its direction lease:
  terminal half-close admission succeeds without I/O, and terminal release racing
  an unfinished half-close satisfies cleanup without replacing received status.
  Published native results still win; retain occupied slots until late I/O ends.
  Derive this behavior from locked runtime admission, not generated error suppression.
  Conversion, observation and service recovery stay on the
  caller stack. Relay active native-hook panics; contain and diagnose late ones
  without panic contents or late request-state mutation. Keep
  `grpc/tla/message_context` and its mounted cancellation/recovery tests aligned.
- gRPC error status and details share one runtime classification in
  `grpc/error_contract.go`. Follow transparent single-cause wrappers, stop at
  an explicit ServiceError, GRPCStatus or recognized designed contract, and
  classify independent join branches separately. An aggregate keeps only a
  unanimous status; otherwise it uses Unknown. Generic public fault details
  never borrow branch-specific names, history or retry traits. Do not use
  tree-wide `errors.As` or `status.FromError` to select a response owner.
  Keep `grpc/tla/error_contract` paired with direct runtime and compiled
  unary/streaming declared-error tests. Raw context cancellation/deadline
  sentinels are native termination outcomes after explicit owners, producing
  canonical context messages without fabricated fault details. Clients decode recognized details
  and otherwise preserve the received error; never replace it with a synthetic
  fault or the local context's error.
- `pkg.ResolveErrorMessage` owns default public message selection and reports
  whether an explicit safe override selected it. `pkg.ErrorSafeMessage` exposes
  the message alone for transport emitters.
  A direct `SafeMessage` overrides named public error text; transparent
  single-cause wrappers add no message. Independent joins without an explicit
  outer owner use `internal server error`. HTTP detail and gRPC default
  status/detail/history consume this owner; history masking uses the returned
  override flag, never a separate remedy traversal or message-text comparison.
  Nil, empty, or code-only outer remedies do not hide an intermediate safe
  owner. JSON-RPC emitters already
  use the message projection.
  Preserve the original diagnostic error and cause. `grpc.NewStatusError` and
  explicit `GRPCStatus` remain author-owned protocol messages; designed typed
  bodies remain public authored data. Keep `pkg/tla/error_disclosure` aligned
  with runtime and generated wire regressions.
- Generated transport observability must remain dependency-free and must never
  emit bodies, params, tool arguments, credentials, or result payloads.
- `http.DecodeResponse` owns original response-body cleanup, bounded restoration
  and error composition for generated HTTP/JSON-RPC decoders and notification
  responses. Generated callbacks own typed/status decoding only. Raw/file success
  transfers the original body; failed metadata validation closes it. Never add a
  second endpoint close or discard a close/read error. The response-body model
  under `http/tla/response_body` records ownership and joined-error invariants.
- Keep HTTP, gRPC, and JSON-RPC semantics distinct unless a deliberately shared
  transport core owns the behavior.
- Keep deployment-owned HTTP policies composable at generated method fields.
  Request body policy must cover built-in JSON, text, form, and multipart
  decoding while exposing the same bounded reader to raw-body services.
  Response-cookie policy may alter only deployment-owned `Domain`, `Secure`,
  and `Expires`; designed cookie attributes remain immutable. Strict response
  negotiation and derived HEAD remain opt-in.
- Canonicalize implicit service collections after validation and before
  generation. Preserve explicit `Server.Services` order; do not let Go package
  initialization order churn aggregate clients, CLIs, or transport contracts.
- A union HTTP or JSON-RPC body gets its own Go transport types. `expr`
  suffixes the branch types of the body, as for object bodies. The HTTP code
  generator renames the union in its private transport IR copy to
  `<Endpoint>RequestBody`, `<Endpoint>StreamingBody` or
  `<Endpoint>[<Status>]ResponseBody` (`nameUnionBodies`). It collects union
  types from that same IR (`collectHTTPUnionTypes`) and visits each body user
  type once by its hash, which names its Go type, not by its identifier: the
  body types of a result type, such as the response body and the element of
  a collection of it, share the identifier and hold different branch types.
  The layouts of the body types (`recordUserTypeLayout` and the
  `JSONPresenceTypes`, `PresencePointerTypes` and `PresenceUseDefaultTypes`
  maps of `codegen.AttributeContext`) are keyed by the same hash, because the
  request and response body types of a result type share its identifier and
  have different layouts. OpenAPI keeps the service union names, so do not
  rename the union in `expr` or in the shared IR.
- `internal/httpencoding` owns named URL field serialization selection and
  source-format matching. Named query maps and named form objects/maps use one
  JSON document; scalar and repeated-scalar formats remain unchanged. Whole-query
  and root-form layouts remain separate boundaries. Transport analysis, runtime
  encoding, OpenAPI and import admission consume this owner; reject mismatched
  source serialization instead of silently importing it as a different codec.
  JSON subtrees use existing Optional/nullability carriers and shared transforms,
  while their outer form keeps native fields. Apply registered physical layouts
  at named collection conversion boundaries as well as object helpers. Keep the
  serialization model and mounted nested-null/import round-trip tests aligned.
- The HTTP code generator declares the request and response body types, and
  the user types nested in them, from the normalized bodies of its transport
  IR (`transportir.Request.Body`, `Request.StreamingBody`,
  `ResponseStatus.Body`). The union types and the transforms use the same
  bodies. The normalization replaces an alias user type with its underlying
  type and a named union user type with its union, so a type declared from
  the evaluated `HTTPEndpointExpr.Body` or `StreamingBody` refers to types
  that the generated code does not declare. The WebSocket payload constructor
  keeps the name of the evaluated streaming body user type
  (`initWebSocketPayloadConstructor`), and `Stream.RequestMessage` keeps the
  evaluated body for the response contract message names.
- Allocate endpoint body names before the HTTP type and layout registries
  (`nameBodyTypes`). Reserve nested types and every preferred body name before
  assigning collision suffixes; retain one allocation for copies of a body.
  Keep this renaming in the Go transport IR so OpenAPI names stay unchanged.
  Named scalar request wrappers also need an endpoint-owned expression ID,
  and the expression copy key must distinguish transport wrappers from
  authored types structurally: an authored name can equal any wrapper ID
  string. Copies preserve the private wrapper marker, and example memoization
  uses it to keep authored and wrapper values separate. The public example
  cache API retains authored-type IDs. The models in
  `http/codegen/tla/body_type_names` check copy, example and naming boundaries.
- HTTP/JSON-RPC conversion constructors share one package owner:
  `ServiceData.finalizeConstructors` runs after wire and CLI declaration analysis.
  Its server/client registries reserve other package declarations, then intern
  complete `constructorPlan` declarations (parameters, result and emitted body).
  Rendering uses that same plan; matching signatures or target types alone do
  not establish conversion identity. Collect constructors independently of type
  deduplication. Rebuild response encoding plans after allocation because they
  retain complete call expressions. FileData import reanalysis must retain the
  allocation relationships. Keep the constructor-ownership model, package
  inventory, selected-field runtime and client/stream/import controls aligned.
- Declaration and helper identifiers use the allocated concrete name from
  `GoValueTypeName`; nullable `GoTypeName`/`GoTypeRef` expressions remain use-site
  storage references. Projected declaration construction normalizes local inputs
  to the intrinsic named declaration without mutating the occurrence graph.
  Selected-view validator definitions and calls use `NamedValidationFunction`
  for their shared identity. Service, view and HTTP named union collectors retain
  their existing hash/cardinality and client/server membership owners. Keep the
  [projected declaration identity model](../../../codegen/service/tla/projected_declaration_identity/README.md)
  paired with both-order identity tests and generated transport/view controls.
- Shared service view validation retains the occurrence carrier separately from
  the intrinsic named validator parameter. `viewValidationCarrier` owns field
  and element presence, concrete access and call arguments. Occurrence Nullable
  unwraps before an ordinary helper; intrinsic helpers accept their wrapper and
  skip absent/null values. Required root, field and collection-position admission
  belongs at the call boundary. Root nullable bodies unwrap before field access;
  diagnostic paths remain the caller's path. Keep viewed metadata envelopes
  non-null independently of nullable Projected payloads and derive their refs via
  NameScope. Method occurrence nullability also participates in viewed-envelope
  reuse and constructor identity; canonical projected declarations stay shared.
  Preserve the occurrence on Projected and adapt canonical projection calls in
  both directions. Transport validators use the recorded `Validate.Name`.
  Projection entry nil guards apply only to physical pointers; shared
  nullable conversion preserves absence/null. Keep the selected-view carrier model
  and complete ordinary/occurrence/intrinsic/root-occurrence HTTP/JSON-RPC runtime matrix aligned.
- A named union, a user type whose attribute is a union, is declared as the
  union type itself under the name of the user type: in the service package
  (`collectUnionTypes`), as a streaming payload or result
  (`buildStreamAttributeData`), and as a projected type in the views package,
  where it takes the `<Name>View` name (`collectViewUnionTypes`). Never emit a
  type definition for the user type. Key union declarations by the referenced
  type's hash, as `NameScope` does, rather than the underlying union's hash:
  raw constructor unions and promoted block branches may share an underlying
  union while referring to different Go declarations. The model under
  `codegen/service/tla/union_declarations` checks collection order and reuse.
- Create promoted block branches with `RootExpr.NewUnionBranch`. Their
  private definition identities prevent DSL copies from merging definitions that
  have the same preferred name. `RootExpr.Prepare` walks source attributes by
  stable paths, reserves authored names and all branch base names, and assigns
  one final name to every copy of a definition before transport preparation.
  Copy memo keys separate private identities from authored string IDs; do not
  encode temporary identities as reserved-looking names. Walk types by pointer
  when evaluating their DSL. Do not let declaration order enter generated
  identity. The model under `dsl/tla/union_names` checks copy identity,
  reservations and ordering. Extend the source-root collector when adding
  a design position that can contain user-authored types.
- Reserve raw union names in the views scope after projecting their branches
  and before rendering enclosing result conversions. Qualified references
  reuse that allocation; they cannot safely allocate names for another package.
  The model under `codegen/service/tla/view_union_names` checks this ordering.
- Copies of union branch attributes must keep their canonical type references
  without traversing unfinished definitions. Use the private DSL helper
  `copyAttributeKeepingType` for named unions, promoted branches and view
  attributes. It isolates metadata entries, validation records and docs while
  preserving type identity. Union branch cycles, including cycles through
  collections, are rejected during design validation because their transforms expand inline; recursion
  through object fields remains supported. The bounded copy and expansion
  models under `dsl/tla/union_copy` reproduce both failures and check the copy
  ownership and cycle-rejection rules.
- `expr.Project` owns recursive selected-view semantics through ordinary
  wrappers, arrays/maps, collections and union branches. Each nested result
  selects its authored view or its own default independently of its parent.
  `projection_conversion.go` binds detached selected graphs to complete physical
  carriers; never prune shared physical declarations or invent Go type names.
  Lower projected carrier package placement before allocating names or taking
  detached conversion snapshots. Canonical service types retain authored package
  locations. Keep the carrier-location model, recursive reference tests and
  custom-package compilation checks aligned with that preparation order.
  Keep one `GoTransform` renderer. Its helper allocator must retain semantic
  source/target identity through `DupAtt` and use the same name for calls,
  declarations and recursion memoization. Preserve the different physical
  layouts of named pointer carriers and anonymous objects. Keep
  `codegen/service/tla/recursive_projection` aligned with these boundaries.
- The shared service-to-view constructor owns the output contract: reject a nil
  object, select and project the view, then run its generated validator before
  returning a result. Unknown views and violated constraints become Loom faults
  wrapping the validation cause. Keep this in `buildViewedResultInit` and its
  renderer so endpoints, interceptors and stream senders share the boundary.
  Object projection helpers preserve nil so collection validation can reject
  non-nullable nil elements without a panic. Nil and empty result collections
  remain valid empty collections. Do not validate excluded fields or reclassify
  errors returned by services or client-side conversion. See the
  [output contract model](../../../codegen/service/tla/result_output/README.md).
- HTTP SSE and WebSocket senders select the response-body constructor from the
  validated projected result view through `writeServerStreamBodySelection`.
  SSE retains the complete response-body family. Mapped data membership comes
  from that selected body's finalized value plan; an omitted field uses its
  existing codec's empty representation without restoring a required/defaulted
  canonical field. Keep the selected-view body model and generated controls
  aligned; sequential view changes are distinct from concurrent SetView safety.
- The JSON-RPC servers build the response body of a viewed result with the
  constructor of the body of its view (`viewedResultBodyInit` in
  `jsonrpc/codegen/stream_viewed_result.go`), as the HTTP response encoders
  do. The unary HTTP handler renders the view that the endpoint returned and,
  when the design fixes no view, names it in the `loom-view` header that the
  client reads; the streams first project the result with `NewViewed<Type>`.
  Stream envelopes take `loom_view` from that same projection. Both endpoint
  and connection senders carry it on notifications and success responses;
  errors omit it. Dynamic SSE streams sample view selection under their mutex.
  Viewed stream clients reuse the projected response decoder and validate each
  frame independently, reject fixed-view mismatches, and fall back to fixed or
  default selection only when the marker is absent or empty. Preserve ordinary
  non-viewed message bytes. Keep the bounded models in `jsonrpc/codegen/tla`
  paired with generated wire, decoder, and raw GET listener tests.
- The server endpoint of a method whose result is a result type returns the
  viewed result, while the result accessors of interceptors take the result
  type. The server wrapper of an interceptor with result access
  (`renderViewedResultInterception`) converts the endpoint result with
  `New<Type>` for the interceptor and projects the returned result with
  `NewViewed<Type>` and the view of the endpoint result, or the view of the
  design. Other interceptors and the client wrappers pass values through.
- HTTP and gRPC client-streaming endpoints return nil after the service sends
  its final result through `SendAndClose`. Reject unary result access there and
  allow streaming result access for those final responses. JSON-RPC WebSocket
  client-streaming service methods return a canonical result per request; keep
  their ordinary result boundary and reject final-response streaming access. Preserve dynamic `SetView` on generated server interceptor
  wrappers. Intercept canonical values at send/receive operations; do not wrap
  the nil endpoint return as a viewed result or buffer final sends. Check the
  lifecycle model in `codegen/service/tla/FinalResultInterceptor.tla` and pair it
  with compiled server/client tests of normal, blocked and failed operations.
- Untagged branch validation receives the actual physical attribute context and
  view mode from its ordinary or projected builder. `codegen.ValidationCode`
  owns nil guards, dereferences and requiredness; never construct an ordinary
  context inside the shared branch helper or add predicate-specific patches.
  Keep `codegen/service/tla/union_validation_context` aligned with ordinary,
  projected, defaulted and selected-view controls. Optional recursive traversal
  is a separate owner and is not established by this context model.
- Named projected object carriers keep required unions as values and other
  fields as pointers or collections. `projectionCarrierValidation` normalizes
  requiredness in the detached conversion graph to match that physical layout,
  so union access stays direct and other reads stay guarded. Anonymous objects
  retain physical requiredness and tags because those affect Go type identity.
- Shared `GoTransform` conversion into pointer carriers preserves absent object
  fields, named scalar pointers and collections independently of full-type
  requiredness. A selected view may legitimately omit them, and its validator
  runs after transport-to-view conversion. Do not materialize missing required
  collections in that carrier or dereference required named scalar pointers.
  Value-target default application and required-array encoding remain separate.
  Keep this decision in the shared transformer, used by unary and WebSocket
  response constructors, rather than adding transport-local guards. The bounded
  [presence model](../../../codegen/tla/projected_presence/README.md) records the
  panic and collection-presence counterexamples and its assumptions.
- Shared service analysis collects custom error types from evaluated HTTP and
  JSON-RPC endpoint mappings as well as service/method declarations. Inherited
  mappings can retain an API error type even when the local error uses the
  default type. Collect mapped custom types before payloads to avoid duplicate
  declarations when they share a type, and retain default constructor precedence.
  Package-name allocation and self-import validation traverse those same mapped
  types. Error descriptors and HTTP body constructors resolve the evaluated
  mapped type's location, which can differ from the method error of the same
  name. Do not emit unused API errors or errors replaced by local mappings.
- Explicit body DSL functions inherit from the full source attribute, including
  inline payload requiredness. `AttributeExpr.Inherit` copies requirements only
  for selected target fields; request-body validation reads `AllRequired()` so
  named payload requirements are included. Before finalization, `AllRequired`
  resolves Reference and Extend requirements with cycle protection: references
  select only existing fields, while extensions include inherited fields. Do not
  copy transport-only required names into a body schema.
- A method result that customizes the requiredness of a result type, such as
  `Result(RT, func() { Required("x") })`, is a renamed copy that keeps the
  identifier of the result type. The service analysis keys projected and
  viewed result types by identifier and name (`projectionKey`), so the result
  type and each copy get their own projected type, viewed result type and
  `NewViewed<Type>` constructor. `ViewExpr` retains authored required/optional
  overrides separately from effective validation. Recompute view requiredness
  from each customized parent before projection and after finalization. Copies
  own view metadata, validation and override slices; preserve nil views until
  collection DSL evaluation inherits element views. Derived HTTP response views
  must preserve authored overrides too. A projected HTTP client body carries the
  projected validation on both the outer attribute and its result type so field
  declarations and conversions use the same requiredness.
- Method-level `Extend` on a named payload or result, including streaming
  positions, must give the copied type its own name. Store inherited bases on
  the copied user type attribute so emitted fields and requiredness share an
  owner. Plain uses must retain their original shape in either traversal order.
  Compose inheritance before validation; create automatic default result views
  from the effective shape during finalization. Preserve explicit view selection.
- Keep requiredness and nullability orthogonal. `expr` owns semantic
  nullability; shared service models use `loom.Nullable[T]` for null-admitting
  object fields; JSON decoding boundaries alone use `loom.Optional[T]` for
  optional non-null fields. Reject nullable shapes at transports that cannot
  preserve absent, null, and concrete states instead of weakening the service
  contract.
- gRPC nullable admission inspects complete service contracts before transport
  finalization and recursively pairs local message/metadata declarations with
  source-owned types. Preserve authored local descendants and scalar wrapper
  intent without mutating either graph. The shared positional predicate admits
  nullable Any only at message roots/fields; collection/union positions and
  metadata do not inherit that exception. Keep both ordering and local-traversal
  models in `expr/tla/grpc_presence_admission` aligned with complete DSL and
  generated Any roundtrip controls.
- Preserve custom scalar conversion metadata and allocated raw/value locals
  when deriving HTTP payload constructor arguments. CLI custom string fields
  parse raw flags with UnmarshalText, validate the raw text, and retain optional
  pointer semantics. Request encoders format custom values consistently with
  path builders through fmt.Sprint; custom types own their wire representation.
- JSON-RPC `Server.ServeHTTP` is an explicit dispatch method that consults the
  configured handler chain at request time. Mount must bind that server method,
  not a promoted method on the embedded handler. `Use` may run before or after
  mounting, but configuration must finish before serving requests. See the
  [handler dispatch model](../../../jsonrpc/codegen/tla/handler_dispatch/README.md).
- Ordinary and custom multipart request decoders share `requestDecoderPartials`.
  `RequestDecodePlan.ElementsMustValidate` owns bound-field error requirements;
  each enclosing wrapper declares its accumulator and barrier. The outer custom
  path retains its body-error and Basic responsibilities. Section construction
  supplies error-return arity; partials must not assume a payload return or add
  a second mapping/conversion policy. Keep callback-before-elements-before-Basic
  ordering and `http/codegen/tla/multipart_decode` aligned with generated wire
  tests, including optional, collection, credential and custom-scalar bindings.
- `RequestDecodePlan` allocates distinct URL-capture, JSON-RPC params-byte and
  query locals after authored field names. HTTP, multipart and JSON-RPC decoder
  renderers consume those names; never reuse a protocol local for another wire
  representation or bypass typed path bindings to avoid a collision.
- Generated request decoders derive root nullability from the evaluated body
  and pass `http.WithNonNullableBody` to decoder factories for non-nullable
  bodies. The built-in JSON decoder rejects root null before unmarshalling;
  optionality still controls EOF independently. Custom decoder factories own
  their codec contract.
- Projected collection view validation checks nil elements before calling the
  element validator. Reject non-nullable elements with an indexed error and
  skip validation of nullable nil elements.
- JSON decoding types use the existing `loom.Optional[T]` carrier for non-null
  array elements and map values. Its decoder rejects null before conversion can
  erase it, including viewed HTTP/WebSocket responses. Authored nullable elements
  retain `loom.Nullable[T]`. Invalid null elements are decoding errors: HTTP
  clients preserve the JSON semantic cause, HTTP requests use the safe `decode_payload`
  problem contract, and WebSocket receivers preserve the decoder error. Keep
  selected-view requiredness in view validation; do not add parallel per-view
  null scanners. See the [collection decoding model](../../../http/codegen/tla/collection_decode/README.md).

## Dependency Baseline

Follow the latest-stable dependency policy in `AGENTS.md`. The root `go.mod`
selects Go and library versions; `Makefile` owns the linter and protobuf-tool
pins; the Node contract job pins Node in `.github/workflows/test.yml`.
Use the stable golangci-lint bundle for Staticcheck in both handwritten and
generated code; `.golangci-generated.yml` enables analysis of generated files.
Do not restore a separate Staticcheck installation or old-Go analyzer workaround.
Keep nested fixture modules and documented external-tool recipes aligned when
updating these owners. Do not reintroduce old-version matrices or compatibility
flags such as `go mod tidy -compat=1.17`.

Use `go.yaml.in/yaml/v3` for Loom-owned YAML serialization. The OpenAPI importer
uses `github.com/pb33f/go-yaml`, the native node API of the current stable
libopenapi release, for both source inspection and parser model nodes. Do not
convert through abandoned YAML modules or retain the YAML v4 release candidate.
The SSE correctness pin and its upstream return condition are documented below.

## Verification Strategy

Follow the validation scope rules in `AGENTS.md`. Ticket acceptance uses direct
regressions, affected packages and a bounded selection of representative
generated-output and transport cases. Keep the full exported-design corpus for
separately chosen audits after major cross-cutting changes; it is not a ticket
review or commit prerequisite. Preserve valid evidence and rerun only checks
invalidated by later changes. Do not multiply every catalog entry across
revisions and isolated runs as a routine ticket gate.

Use the [formal model index](references/formal-models.md) alongside direct tests
for ownership, ordering and semantic projection changes. New weaknesses require
counterexample/model and correspondence-ledger updates, affected claim review,
audited checks and independent re-review before dependent implementation relies
on the result. A passing model does not establish generated-Go correctness.
Keep sources, configurations, theorem manifests, toolchain pins, run instructions
and concise findings; follow the index's cleanup rule for task-owned compiled
and checker output after validation and review, preserving active dependencies.

For framework and codegen bugs, add the failing direct test first. Pair the test
layers that apply:

1. direct seam tests for `expr`, shared IR, generator sections, or runtime logic
2. structural assertions for emitted contract/code shape
3. rendered or golden coverage
4. compile-after-generation coverage
5. relevant integration and adversarial transport cases

OpenAPI changes require:

- direct structural assertions
- meaningful rendered specimens under `http/codegen/testdata`
- exact-byte comparison across process-isolated generations when output
  determinism is part of the contract; same-process or semantic comparisons do
  not exercise independent map seeds
- deterministic nested serialization from every custom marshaler that can
  contribute map-valued schema, example, extension, or reference content
- JSON v2 lint coverage that rejects direct `json.Marshal` calls inside
  `MarshalJSON` methods unless they pass `json.Deterministic(true)`
- `libopenapi` parsing
- Vacuum linting with the pinned contract rules and invalid-spec controls
- downstream consumer smoke generation where applicable

Serialization migrations must preserve error identity as well as successful
decoding. At JSON request boundaries, test empty, JSON-whitespace-only,
malformed, truncated, and valid bodies through both the handwritten runtime and
generated required and optional body handling. At the generated transport seam,
assert HTTP status, problem code and detail, service invocation, and decoded
payload state.

Keep SSE request-ID names in DSL form in the transport IR. Resolve
`SSEData.RequestIDField` with `GoifyAtt` at the Go generator boundary, using the
payload attribute metadata; retain the separately evaluated pointer semantics.

The SSE library owns framing, parsing, event limits and event memory. Shared
`http.SSEStreamReader` adapts its iterator with serialized reads, context-triggered
body closure and serialized iterator release. HTTP and JSON-RPC clients consume
parsed events directly; do not reintroduce framing or reparsing in generators.
EOF discards an incomplete block after delivering preceding complete events.
See the [framing and lifecycle models](../../../http/tla/sse_framing/README.md).

Temporarily use `github.com/CaliLuke/go-sse` at the pinned version in `go.mod`.
It contains [upstream PR #61](https://github.com/tmaxmax/go-sse/pull/61) plus only
the module/import path changes needed for transitive Go consumers. The upstream
PR branch remains separate from the packaging branch. When an upstream revision
contains the correction, switch runtime imports and module requirements back to
`github.com/tmaxmax/go-sse`, tidy affected fixture modules and rerun the focused
SSE reader/transport checks. Do not use a transitive `replace`, vendor a parser
patch, or add local EOF filters.

SSE changes require coverage for:

- happy paths
- pre-stream endpoint failures
- protocol-level error event compatibility
- compile-after-generation
- branch-specific connection timing

Pulse (`pulse/pool`, `pulse/streaming`, `pulse/rmap`) tests run against
miniredis by default, so `make test` stays hermetic. miniredis does not model
every Redis behavior: it lacks the Lua `struct` library (the test helper
prepends a shim), keeps pending entries of deleted stream ids as Redis 6.2
does, and accepts commands that older servers reject. Changes to pulse Lua
scripts, stream, consumer-group, or expiry commands require the real-Redis
tier:

- Pulse supports only the latest stable Redis release, currently 8.10.2.
  `make test-pulse-redis` starts the image pinned in `scripts/test_pulse_redis.sh`
  with Docker and runs `./pulse/...` with `-race`. GitHub CI uses this same
  runner and pin. Update the pin and public guidance when adopting a newer
  stable release. `LOOM_PULSE_REDIS_ADDR=host:port` selects an existing server.
- The shared helper is `pulse/internal/redistest`. Each package uses its own
  database and flushes it per test, so point the tier at a disposable server
  only. It refuses a non-loopback address unless
  `LOOM_PULSE_REDIS_ALLOW_REMOTE=1`, and the Makefile unexports
  `LOOM_PULSE_REDIS_ADDR` for every other target. Pub/sub is server-wide, so
  packages run with `-p 1`. `ci-local` excludes this tier because pre-push must
  not need Docker.
- Keep miniredis limitations explicit with `Server.Real()` assertions. They
  do not represent support for older Redis versions. Tests that need miniredis
  clock control need a wall-clock path on the supported real server.
- Pool protocol 2 ownership lives in plain Redis hashes (`owners` and
  `owner-epochs`). Claim and release scripts update ownership, job indexes,
  and payloads together with rmap revision notifications. Never perform a
  per-job write without checking the owning worker and epoch. Cleanup checks
  its lock token and the Redis-time heartbeat in the same script that releases
  and requeues jobs. Lost release replies stay pending locally; never restart
  a handler on the assumption that Redis rejected its release.
- Keep heartbeat I/O and resume verification independent of the lease monitor.
  Redis operations may hold the ownership lock but never the handler lock
  needed by fencing. Distinguish a changed owner/epoch from an expired lease
  whose ownership record remains; the latter must retain paused work for retry.
  Acquire the ownership lock before the handler lock when both are required.
  Record renewal request start times, never delayed reply times. Resume must check
  the current owner, epoch, registration, and Redis lease without creating
  ownership. A fenced handler already stopped successfully must not receive
  another Stop during release or eviction.
- Reconcile job placement periodically as well as on worker-map updates.
  Membership can arrive before a joining heartbeat or an already-routed start;
  heartbeat notifications alone miss the latter case. Keep the bounded
  `pulse/pool/tla/JoinRebalance.tla` model and watcher regressions paired with
  real-Redis migration coverage when changing these triggers.
- Close all protocol 1 nodes before migration. Join registration and backfill
  are atomic with the initial node heartbeat; routing and claims reject live
  incompatible nodes. Preserve epoch counters across pool shutdown and read
  incremented tokens as decimal strings, not Lua floating-point values.
- `pulse/pool/redis_semantics_test.go` pins the stream behaviors the pool
  scripts rely on: single-id `XPENDING`, `XINFO GROUPS` fields, `XADD MAXLEN ~`
  in Lua, pending entries of deleted ids under `XAUTOCLAIM`, and script
  atomicity under concurrent claims.

Use repository gates rather than duplicating their logic:

```bash
go fmt ./...
make lint
make test
make coverage-ratchet         # protected consumer-aware boundaries
make openapi-contract          # OpenAPI work
make generated-code-quality    # generated Go/output work
make test-testdata-compile     # all exported design fixtures: gen, example, build, vet
make integration-test          # transport behavior
make test-pulse-redis          # pulse work, pinned latest stable Redis (Docker)
./check.sh --full              # full repository verification
```

The checked-in floors in `coverage/baselines.json` protect evaluated-design
semantics, shared service codegen, HTTP/gRPC/JSON-RPC codegen, and the shared
OpenAPI pipeline. `make coverage-baseline` rewrites them from the current
consumer-aware profile. Treat any reduction as an explicit reviewed contract
change and document its reason; aggregate coverage remains an inspection
artifact, not the ratchet.

## Local and Remote Source Modes

HTTP and JSON-RPC temp-module tests share a worktree-local source selector:

```bash
make loom-local
make loom-remote
make loom-status
```

- Use local mode, or set `LOOM_DIR=/absolute/path`, while testing unpushed
  framework changes.
- Use remote mode for pushed-commit reproducibility.
- The source-mode marker is untracked worktree state; do not commit it.
- A pre-push check cannot fetch the commit being pushed for the first time. Use
  local mode for that push, then restore remote mode afterward.
- External temp modules must pin a pushed GitHub commit in reproducible CI.

Temp-copy regeneration tests for checked-in fixtures are intentionally
local-only and should rewrite the copied fixture's `replace` directive to the
current repository root before invoking `loom gen`.

## Exported Design Compile Corpus

`make test-testdata-compile` discovers exported zero-argument design functions
and function-valued variables from the five testdata packages (service, HTTP,
gRPC, JSON-RPC, and expr). It uses `loomsource.Resolve`, runs `loom gen` (including
protoc) and `loom example` in isolated temporary modules, then tidies, builds and
vets the complete module. Cases share Go caches; `TESTDATA_PARALLEL` bounds
concurrent modules (default 2). `TESTDATA_RUN` selects a domain or design.

The ordinary unit tier checks discovery and the expectation manifest; the full
corpus is a separate CI matrix and part of `ci-local`. Read
`internal/testdatacompile/README.md` before adding an expected failure. Never
accept an unexpected failure by silently skipping a design. Expected validation
errors need an exact case and diagnostic; generator/build failures need a GitHub
issue. A fixed case must have its stale expectation removed.

## Integration Process Ownership

HTTP, JSON-RPC and exported-design compile harness subprocesses use
`internal/testprocess`. On Unix, a separate process-group guardian observes a
parent-owned pipe, so test timeout
panics and parent termination clean up servers, CLIs and build subprocesses.
Do not bypass the wrapper execution methods or detach descendants from their
process group. JSON-RPC CLI calls also enforce a 30-second deadline. The
`internal/testprocess/tla` model records the ownership protocol and its bounds.
Windows retains standard `os/exec` behavior; integration suites remain Unix-only.

## Fixture Contracts

Treat these as regression surfaces, not demos:

- `http/integration_tests/fixtures/ticktock`
- `jsonrpc/integration_tests/fixtures/ticktock`

The checked-in JSON-RPC ticktock fixture covers POST-initiated SSE. The raw
`events/stream` GET listener is covered by
`jsonrpc/integration_tests/tests/sse_get_listener_test.go`, which regenerates a
temporary listener variant. Extend the test matching the branch being changed.

## Documentation Ownership

- Public user behavior belongs under `docs/`.
- Update the consumer `loom` skill only when users must design, generate, wire,
  or reason about their applications differently.
- Framework architecture, contributor workflow, source-mode behavior, and
  internal invariants belong in this skill or `AGENTS.md`, not the consumer
  skill.
- Active framework plans belong under `roadmap/`; remove obsolete temporary
  plans when their durable guidance has moved to docs or skills.
- Update `references/repo-map.md` when framework ownership or navigation
  changes.

## Completion

Before handoff:

1. format touched Go files with the selected Go 1.27 toolchain
2. run proportionate direct and broad gates
3. regenerate every intentionally affected checked-in fixture
4. inspect `git diff --check` and `git status --short`
5. preserve unrelated and worktree-local files
6. commit and push unless the user explicitly asks to stop short

When the change affects public usage, report both the implementation location
and the updated user-facing guide. For internal-only work, do not burden the
consumer skill with contributor detail.

## References

- `references/repo-map.md`
- [Formal models and proof maintenance](references/formal-models.md)
- repository `AGENTS.md`
- `roadmap/`
- `docs/`

Response observation, clue HTTP logging and Debug share `transport.CaptureResponse`.
Capture state is separate from the writer; `httpsnoop.Wrap` owns conditional
interface forwarding. Loom hooks own status commit and byte-count policy,
including informational responses, upgrades, flushes and I/O fast paths. Do not
add transport-local forwarding wrappers. Debug opts into bounded body retention
through WithResponseBodyCapture; ordinary metrics consumers retain no body.
WithResponseHeaderCapture retains a detached first-commit snapshot only when
headers remain stable across forwarding. Opaque first ReadFrom and changed
metadata are unavailable; Debug omits ambiguous body previews. Do not infer a
wire commit from source reads or reuse final mutable headers. Keep metadata
ordering counterexamples and runtime controls aligned in the response model.
Capture only accepted bytes, trim ReadFrom lookahead to its returned count, and
discard oversized previews with an explicit omission marker. Body snapshots are
detached and synchronized; writer concurrency remains the caller's responsibility.
Request and client response read observation lives in `transport.CaptureBodyReads`; it shares the
private bounded-retention primitive with response capture but forwards each Read
and Close unchanged. Never hold its snapshot lock across underlying I/O or drain
to finish a diagnostic. Debug snapshots at handler return, marking non-EOF
captures incomplete. Downstream request-size-policy probe reads count as consumed
bytes when Debug wraps that policy. Nil and http.NoBody stay unchanged. Keep the
[request observation model](../../../observability/transport/tla/request_capture/README.md)
aligned with errors, concurrent Close and unfinished-upload runtime controls.
Client Debug retains exchange-owned request/response metadata and immutable request
preview, and observes response bodies with this same read owner. Completion logs
once per exchange at EOF or caller Close, never via mutable latest-request state.
`Fprint` only snapshots; partial structured previews are omitted. Close forwards
first and may snapshot before an in-flight read returns. Do not add draining or
response replay to make diagnostics appear earlier. Keep `http/tla/client_debug_response`
and its timing/error/redaction tests aligned. Client request prefix/GetBody capture
remains a separate existing policy.
Diagnostic masking lives in `observability/diagnostics`, shared by client and
server Debug and clue access-log URLs. Its constructor copies public field sets
into an immutable policy. Access-log options snapshot at option construction and
compose immutable policies without mutating requests. The shared HTTP example
base selects generated Fields for the current server's HTTP and JSON-RPC services;
the JSON-RPC extension inherits that configuration. Do not duplicate credential
inference or add policy separately to each transport's example renderer.
Project credential source identity from `HTTPEndpointExpr.CredentialBindings`,
including explicit duplicate wire operations; never re-infer security in a
transport renderer. Prepared body fields derive JSON/form names from the actual
representation tag owner. Emit service metadata once for application wiring and
selected example CLI policies. Whole-body/path omission, route-variable masking,
and unsupported credential body codecs must not leave alternate previews leaking.
Keep the [redaction model](../../../observability/diagnostics/tla/redaction/README.md)
aligned with delayed completion, custom mapped roles, wire preservation and
ordinary visibility tests. No metadata is emitted for ordinary/disabled services.
The callback-scoped httpsnoop metrics
API does not match the independently started/ended request observer lifecycle.

## Release Ownership

`internal/release` owns immutable-source publication through
`.github/workflows/release.yml`. The workflow has manual alpha, scheduled alpha,
and explicit stable-promotion entry points sharing one concurrency group.
`make release` only dispatches it. Exact-source CI and the aggregate release
eligibility job replace local release preflight; publication never edits source
or main. See the release skill for retries and completion requirements.

`pkg.Version` derives the Loom module version from Go build metadata; local
checkouts and replacements have the explicit `(devel)` identity. `Major` is the
module API namespace, not a mutable release stamp. Fixture manifests record the
locally built generator identity and retain their design digests. Documentation
pins are curated together against the recommendation in `docs/_index.md`,
independently of release tags. Change `internal/release/train.json` deliberately
when starting the next train after stable promotion.
