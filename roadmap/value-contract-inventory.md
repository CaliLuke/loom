# Value pipeline inventory

Reconciled for #573 at `15a4c467`. The [accepted contract](value-contract-design.md)
remains the semantic authority. This is a source and retained-evidence audit,
not a new generator run or a claim that every consumer has migrated. The initial
inventory was taken at `f5b786b39675e2b5c1466f04f3301b7b337779b6`.

## Method and dispositions

Both inventory searches were rerun, and example/default fields were traced to
production readers. Test files are evidence, not additional production consumers.

```sh
rg -n 'CanonicalizeExample|enumvalue\.Normalize|protoJSONExample|normalizeOpenAPIExample|PayloadEx|StreamingPayloadEx|\.Example\(' expr codegen http/codegen jsonrpc/codegen grpc/codegen internal --glob '*.go' --glob '!**/*_test.go' --glob '!**/testdata/**' --glob '!**/gen/**'
rg -n 'jsonExample|flagDefault|NewFlagData|Example:|DefaultValue:' codegen/cli http/codegen/client_cli.go grpc/codegen/client_cli.go --glob '!**/*_test.go'
```

- **Migrated:** the relevant built-in consumer uses the shared semantic owner.
- **Intentional:** a compatibility API, sampler, location codec or renderer retains
  its own documented responsibility; it is not an alternative semantic owner.
- **Unresolved:** the row identifies a remaining ownership, contract or evidence
  question and links it to the finite follow-up list below.

A file can contain more than one disposition. An unconsumed field does not prove
that computing it is harmless: sampling can advance the scoped random generator
or populate its cache and affect a later emitted example. Public data structures
also need a compatibility assessment before fields are removed.

## Semantic core and service carriers

Paths in this table are repository-relative.

| Production file or owner | Current disposition and evidence |
| --- | --- |
| `expr/example.go` | **Migrated / intentional.** Shared source selection and effective constraints govern the sampler; public `AttributeExpr.Example` retains its raw-value contract. `example_effective_constraints_test.go` and `value_synthesis_test.go` check narrowed constraints and authored-source preservation. |
| `expr/example_length.go`, `expr/types.go`, `expr/types_union.go`, `expr/user_type.go` | **Intentional sampler internals.** Recursive sampling and memoization remain behind the shared synthesis graph; these recursive `.Example` calls are not transport projection. `value_synthesis_test.go` checks retained union choice and absence of authored replacement; `recursive_length_example_test.go` covers recursion/length behavior. |
| `expr/value_synthesis.go` (new inventory match) | **Migrated.** `ValueContext.Synthesize` samples its isolated prepared graph and retains union occurrence/branch identity. It cannot replace an authored failure. `TestValueSynthesisCannotReplaceAuthoredFailure` and `TestValueSynthesisNamedUnionRetainsChoice` are direct controls. |
| `expr/example_canonicalization.go` | **Intentional public compatibility API.** `CanonicalizeExample` preserves legacy JSON-shape/pass-through behavior. `example_canonicalization_test.go` pins it. Its presence does not authorize built-in failure fallback; see R1 and R4 below. |
| `expr/json_schema_inline.go` | **Migrated primary path / unresolved fallback (R4).** Examples/defaults resolve through `inlineSemanticJSONValue`; enum clauses use declared-shape resolution. `TestInlineJSONSchemaUsesResolvedDeclaredJSONValues` covers typed projection. Failed resolution still falls back to `CanonicalizeExample`, and `TestInlineJSONSchema/leaves_ambiguous_union_examples_unchanged` explicitly preserves an ambiguous raw example. |
| `expr/http_body_types.go` | **Migrated body ownership / intentional compatibility helper.** Controlled body copies preserve semantic/source bindings; `http_body_identity_test.go` and `value_source_binding_test.go` cover them. Exported `UnionToObject` still samples/serializes a raw example, but has no production caller in this repository; `TestUnionToObjectUsesTaggedDiscriminators` protects that compatibility behavior. |
| `internal/enumvalue/value.go` | **Migrated projection / intentional fallback boundary.** `Normalize` uses shared resolution, then declared-shape projection. Its documented opaque/invalid fallback is not semantic admission. `value_test.go` checks shared values, excluded root predicates, structs, and borrowed opaque/cycle boundaries. Callers must separately validate or omit fallback values. |
| `codegen/validation_render.go` | **Migrated.** Effective enum clauses own admission; `enumvalue.Normalize` only supplies JSON comparison shape. `validation_collection_enum_test.go` and `validation_composite_enum_test.go` cover generated validation. |
| `codegen/service/service_data.go`, `service_data_methods.go`, `value_data.go` | **Migrated owner / compatibility carriers.** `ValueData` selects and resolves service payload/result/error/stream sources. Raw `PayloadEx`, `ResultEx` and streaming fields come from `LegacyValue`; only `PayloadEx` has a production reader, in the HTTP plain/direct CLI fallback. `value_data_test.go` covers retained values, reanalysis, streams and errors. |
| `expr/resolved_value.go`, `value_source.go`, `value_plan.go`, `value_projection.go` | **Migrated shared API.** These owners carry immutable source/occurrence identity, selection, representation plans and projection outcomes. `value_snapshot_test.go`, `value_resolve_test.go`, `value_occurrence_queries_test.go` and `value_projection_test.go` are direct anchors; [the correspondence ledger](../expr/lean/value_projection/correspondence.md) records the narrower formal claims. |

## HTTP, JSON-RPC and CLI consumers

| Production file or owner | Current disposition and evidence |
| --- | --- |
| `http/codegen/service_data_payload.go` | **Migrated JSON examples / unresolved defaults and location samples (R1, R2).** JSON body arguments consume retained `ClientBody.Value`; map-query/location examples still sample independently. `default_body_test.go`, `optional_value_request_body_test.go`, and both transports' `collection_defaults_cli_test.go` protect existing body/default behavior. |
| `http/codegen/service_data_cli_example.go` (new inventory match) | **Migrated `retainedJSON` / unresolved `cliBodyDefault` (R1).** Retained examples project through the actual body runtime plan with explicit outcomes. Collection body defaults still use `GetDefault` followed by `CanonicalizeExample`, and reach real flag defaults. The non-JSON branch preserves the plain codec; its selection remains part of R2. |
| `http/codegen/service_data_init_args.go` | **Unresolved material auth examples (R2) / unconsumed response metadata (R3).** Basic-auth samples reach CLI flags; header/cookie examples copied into response/error constructors have no internal reader. Existing `client_cli_test.go` covers rendered CLI shapes. |
| `http/codegen/service_data_routes.go` | **Intentional path codec / unresolved unused sampling (R3).** Path builders consume argument names/types/values, not `PathInit` example fields. Actual path-flag examples come from request element data. Path tests protect encoding; they do not establish that removing sampling leaves later random choices unchanged. |
| `http/codegen/service_data_transport_helpers.go` | **Intentional location/default codecs / unresolved example source (R2, R3).** Request path/query/header/cookie examples flow to payload CLI arguments and remain independently sampled. Defaults arrive through effective transport copies. Response header/cookie example fields have no internal reader. `client_cli_test.go` and shared CLI default controls protect current codecs. |
| `http/codegen/service_data_body_types.go` | **Migrated runtime plans / unresolved unused sampling (R3).** Request/response `TypeData.Value` carries the source and plan. `TypeData.Example` and transform `InitArgData.Example` have no internal reader. `value_plan_test.go` checks body/result/error ownership. |
| `http/codegen/service_data_type_registry.go` | **Layout/validation owner / unresolved unused sampling (R3).** `attributeTypeData` still samples into `TypeData.Example`; no production code reads that field. Existing body/layout tests do not prove sampling-order neutrality. |
| `http/codegen/websocket.go` | **Migrated stream plan / unresolved unused sampling (R3).** Stream body construction uses retained `StreamingValue`; the payload constructor's example field has no internal reader. `value_plan_test.go` covers streaming ownership. |
| `http/codegen/client_cli.go` | **Migrated JSON routing/availability / intentional plain fallback.** JSON flags use retained plans, including the direct-body branch. Plain flags retain text/byte behavior; their material source-selection gap is R2. `codegen/cli/cli_test.go`, `usage_test.go` and the #565 generated controls protect diagnostics and omission. |
| `http/codegen/misc_sections.go` | **Intentional renderer.** `renderPayloadExtraction` is a name match, not example selection. It renders supplied argument/type data. |
| `http/codegen/internal/representation/value_carriers.go`, `value_plans.go`, `example_preparation*.go` | **Migrated infrastructure.** Service sources are paired with occurrence-specific runtime/documentation targets; prepared examples retain source selection. `http/codegen/value_plan_test.go` and OpenAPI prepared-example tests cover these edges. |
| `codegen/cli/cli.go`, `payload.go`, `conversion.go`, `command_data.go`, `flag_parsing.go`, `usage.go` | **Intentional codecs/renderers; availability migrated.** `NewFlagData`, `jsonExample` and `flagDefault` format supplied values; they do not select semantic sources. Plain byte/text and JSON collection flag syntax remain distinct. Empty example text denotes an unavailable hint. `defaults_test.go`, `cli_test.go` and `usage_test.go` cover defaults, diagnostics and help. |

JSON-RPC reuses HTTP service analysis and CLI generation, so R1–R3 apply to the
corresponding shared paths. Its own `collection_defaults_cli_test.go` is a
required control for any default change. JSON-RPC intentionally skips HTTP-only
WebSocket body declarations; that is not a missing value consumer.

## gRPC consumers

| Production file or owner | Current disposition and evidence |
| --- | --- |
| `grpc/codegen/service_data_analysis.go` | **Migrated message example / unresolved metadata example (R2).** `protoJSONExample` takes `MethodData.PayloadValue`; request metadata arguments still copy independently sampled examples into CLI flags. |
| `grpc/codegen/service_data_helpers.go` | **Migrated effective defaults / unresolved metadata example (R2).** Metadata defaults use `EffectiveConstraintsFor`; `c.Example` still supplies the material request-flag example. `mapped_metadata_test.go` protects transport mappings, not retained example precedence. |
| `grpc/codegen/service_data_convert.go` | **Unresolved unused sampling (R3).** Converter arguments copy metadata examples or sample source attributes. Renderers consume conversion names/types/code, not those example fields. Their random/cache effects remain to be checked. |
| `grpc/codegen/client_cli_example.go` | **Migrated protobuf projection.** Retained source/member/branch identities flow through allocated protobuf names and wrappers without synthesis. `client_cli_protojson_test.go` covers source precedence, selected/absent unions, wrapper collections, invalid/unavailable outcomes and generated replay. Explicit-null support has a separate evidence boundary (R6). |
| `grpc/codegen/client_cli.go` | **Migrated message codec and omission / intentional metadata codec.** The message flag uses protojson; metadata uses shared CLI formatting. `client_cli_protojson_corpus_test.go` checks descriptor-based decoding; `client_cli_determinism_test.go` checks isolated-process output. |

## OpenAPI consumers

| Production file or owner | Current disposition and evidence |
| --- | --- |
| `http/codegen/openapi/internal/ir/analyzer.go` | **Migrated built-in examples / intentional custom callback.** Raw `attr.Example` runs only for an explicitly installed `WithExampleValue` callback, not ordinary generation. Effective defaults use declared-value projection before target visibility/formatting. Prepared-example and default tests protect these separate paths. |
| `http/codegen/openapi/internal/ir/document.go` | **Migrated built-in route / intentional adapter.** Built-in `initExamples` requires prepared targets. Exported `OpenAPIExampleValue` retains raw compatibility behavior; it is not the built-in source-selection route. |
| `http/codegen/openapi/internal/ir/document_examples.go`, `analyzer_validation.go` | **Migrated effective admission / unresolved untagged enum/default interpretation (R7).** Visibility and wire formatting remain here, but untagged enum/default branches are matched again after declared/wire conversion. Prepared retained examples use their separate migrated path. `document_examples_test.go`, `map_defaults_test.go`, `map_defaults_encoding_test.go` and `untagged_byte_examples_test.go` cover the exercised boundaries. |
| `http/codegen/openapi/internal/ir/example_generation.go` | **Migrated and deleted** in `489d93cf`. Shared `http/codegen/internal/representation/example_preparation*.go` and `ir/prepared_examples.go` replace the private synthesis path. |
| `http/codegen/openapi/internal/ir/prepared_examples.go` (new owner) | **Migrated.** Projects prepared retained values and handles outcomes; no replacement source sampling. `prepared_builtin_examples_test.go` and v3 `shared_value_examples_test.go` protect the ordinary pipeline. |
| `http/codegen/openapi/internal/ir/document_cookies.go` | **Migrated source / intentional cookie codec.** Prepared values retain member identity before cookie text formatting. `document_cookies_test.go` checks the emitted cookie examples. |
| `http/codegen/openapi/v3/example.go` | **Built-in route migrated; old private helper has no production caller.** `initExamples` is exercised only by tests. Actual v3 generation goes through the shared IR and `ir_adapter.go`; `example_surfaces_test.go`, `shared_value_examples_test.go` and `singleton_named_example_test.go` cover rendering and metadata. |

The [#572 acceptance record](../internal/valuecontract/OPENAPI_EXAMPLES.md)
preserves the delivered generated/consumer comparisons. R4 below concerns inline
schemas, not reopening the completed OpenAPI example migration.

## Additional ownership boundaries

| Boundary | Current disposition / evidence |
| --- | --- |
| `expr/random.go` | **Intentional sampler memo.** Scope/cache effects of unused calls belong to R3; the bounded [value-projection model](../expr/tla/value_projection/README.md) does not prove PRNG sequence equivalence. |
| `expr/attribute.go`, `attribute_validate.go` | **Migrated source/admission boundaries.** Inheritance and declaration validation feed the shared owner; `value_source_eligibility_test.go`, `value_source_binding_test.go` and effective-constraint tests cover actual Go extraction. |
| `internal/examplevalue/union.go` | **Intentional compatibility carrier.** Retains an existing selected union at the legacy boundary; it is not permission for a target to reselect. `value_synthesis_test.go` and service `value_data_test.go` check retained choices. |
| `internal/jsonkey/key.go` | **Intentional shared key codec.** Spelling and collisions remain codec responsibilities. #456's [diagnostic correspondence](../expr/lean/value_projection/correspondence.md#map-collision-diagnostics-456) distinguishes known witnesses from unknown/opaque inputs. |
| `internal/valuecontract/production_graph_test.go` and specialized adapters | **Unresolved generalization, explicit tested bounds (R5).** The general adapter rejects duplicate wire aliases, alias-local constraints and named/key enums. Specialized #571 groups cover bounded scalar-key/alias cases; they do not remove those guards. |

## Remaining work

These are follow-up scopes, not a batch implementation authorization. R1 and R2
are visible ownership deviations; this audit does not claim an observed runtime
failure for them. R3–R6 distinguish cleanup, compatibility and evidence questions; R7 records another source-level ownership gap without claiming a reproduced output failure.

### R1 — JSON CLI body default projection

`cliBodyDefault` selects a payload default and canonicalizes it independently,
then `NewFlagData`/`flagDefault` turns it into the default accepted by a real JSON
body flag. This bypasses the retained body representation owner even though the
existing collection-default regressions pass.

Start with one direct table for bytes, union and mapped-key defaults, plus the
existing HTTP and JSON-RPC collection-default controls and a plain-byte default
control. Establish the target representation and preserve absent/default semantics
before selecting a repair. Keep this separate from ordinary location codecs.

### R2 — Material location and metadata example sources

HTTP request path/query/header/cookie/basic-auth, non-JSON/plain body, and gRPC
request metadata examples still sample attributes independently before emitting
CLI help/diagnostics.
Their text, byte, numeric and collection codecs are intentional; their source
selection has not migrated to the retained occurrence owner.

Verify authored root-versus-member precedence and unchanged flag spelling at one
HTTP and one gRPC shared seam, including a plain Bytes body example control.
Select one bounded transport/position follow-up
before implementation. Existing CLI renderer controls can be reused; a new
transport matrix is not required merely to establish selection ownership.

### R3 — Unconsumed example fields and sampling effects

Body/type-registry, response constructor, path initializer, WebSocket constructor
and gRPC converter example fields have no internal production reader. Their
sampling calls can still affect later samples through generator/cache state.
Exported carrier fields can also have external users.

For one carrier class, identify public compatibility requirements and compare the
later material examples with and without its sampling, including repeated and
reordered construction. Only then decide whether to remove the call or populate
the field from its retained owner. Reuse a relevant generated response/stream or
converter fixture if that follow-up changes emitted code. This is cleanup and
ordering evidence, not a demonstrated transport bug.

### R4 — Inline schema failure outcomes versus compatibility

`inlineSemanticJSONValue` and `inlineDeclaredEnumValues` can fall back to the raw
compatibility adapter after semantic resolution fails. The existing inline-schema
regression explicitly keeps an ambiguous union example unchanged, while the value
contract prohibits built-in consumers from bypassing failure outcomes.

Resolve this concrete contract discrepancy before changing public behavior:
the exported `InlineJSONSchema` API has no other production caller in this
repository. Identify which public compatibility guarantees require pass-through
and which schema-generation outcomes must honor semantic failures. Use the existing ambiguous-example control alongside
resolved bytes/union/default controls, then check the emitted example against its
schema. Preserve opaque-codec and enum projection boundaries. This audit records
the conflict; it does not authorize dropping a compatibility guarantee.

### R5 — Reference adapter boundaries

`productionGraph.capture` rejects duplicate source wire aliases and constrained
alias/key shapes that it cannot independently lower. `TestProductionConstraintGuardFailsClosed`
ensures those guards actually reject; `expr/value_alias_ownership_test.go` checks
production alias/visibility behavior. The specialized `named-key-contracts` and
`effective-alias-lengths` groups supply narrower independent comparisons.

Start with the duplicate-alias shape already exercised by the rejection control
and mapped-metadata tests: decide whether a separately scoped independent lowering
is needed for that supported graph. Retain every unsupported-shape guard until
its replacement has independent inputs and negative controls. Arbitrary Go
reflection, codecs, DSL extraction and generated compilation remain tested/external
boundaries; their exclusion is not evidence of a production bug or a mandate to
formalize all of Go.

### R6 — Revalidate historical compiler limitations before assigning repairs

The ledger records earlier failures for collision-free `MapOf(Any, String)` keys,
nullable `Any` in gRPC converters, and named Bytes response validation.
They were not regenerated by #573. Current gRPC map-key validation rejects Any keys in supported gRPC positions, and
`TestProtoBufTransformNullableAnyObjectPreservesPresence` now checks nullable
conversion snippets. Neither fact establishes the current outcome of every
historical generated module.

The collision-free Any-key HTTP companion was subsequently reproduced at
`5c4702ac`: declarations used `map[any]string`, but a conversion still allocated
`map[loom.JSONValue]string`. The shared transformer now uses the existing map-key
renderer for its allocation. `TestTransformMapRendersTargetKeyAndValueTypes`
checks Any keys separately from Any values; `TestGeneratedAnyMapKeyBuilds`
generates, builds and vets the retained HTTP design. This closes that compiler
item without changing gRPC key restrictions or making a general codec claim.

The named Bytes response case was subsequently reproduced at `d781f879` and
repaired in `applyUserResponseBodyTypeData`. Validator arguments now follow the
allocated type reference and body value reference; primitive-root validation
uses the same representation. `TestHTTPDirectBuilderSeams` checks the analysis
contract, and `TestBytesRepresentationGeneratedHTTP` regenerates the retained
`BytesRepresentationDSL`, compiles and vets its module, and checks valid and
invalid responses through the generated client. This closes the named Bytes
response item independently of the Any cases.

For the remaining nullable-Any case, recover the design and transport from the retained
record, then run only that specimen through current validation/generation/compilation. Record rejected,
repaired or still-failing outcomes before creating a fix ticket. Do not turn an
old compiler diagnostic into a claim about current main, or treat a snippet test
as proof that the complete generated application compiles.

### R7 — Retained branch identity for OpenAPI enum/default projection

OpenAPI defaults in `analyzer.go` and enum clauses in `analyzer_validation.go`
pass through `enumvalue.Normalize` into `projectOpenAPIExample` and
`normalizeOpenAPIExampleForAttribute`. Their untagged-union branches call
`matchingUntaggedOpenAPIBranch` on converted values instead of carrying the
resolved branch identity. A missing/ambiguous match returns the value without
branch-specific projection. This is a source-level duplicate interpretation
boundary; the audit has not reproduced a wrong emitted default or enum.

Use one default and one enum control whose semantically selected untagged branch
shares the declared/converted matcher input with another branch but has different
field visibility. Branch-specific visibility projection happens after matching.
Check retained identity, projected values and schema validity. Establish that
boundary before deciding the repair; preserve the intentional raw compatibility
adapter and the already migrated prepared-example path.

## Evidence policy

This reconciliation changes documentation only. It reuses the recorded delivery
commits and tests above; it does not rerun proof/kernel checks, regenerate fixtures,
or refresh historical artifact manifests. A future code change must select checks
for the specific obligation it changes, preserve meaningful negative controls,
and update this inventory and the correspondence ledger. Follow
[the execution plan](value-contract-plan.md) and [repository validation rules](../AGENTS.md).
