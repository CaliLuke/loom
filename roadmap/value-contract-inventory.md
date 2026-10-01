# Value pipeline inventory

Reference: `f5b786b39675e2b5c1466f04f3301b7b337779b6`. This is the initial
migration inventory for [the contract](value-contract-design.md). A future
implementation refreshes it before editing and reconciles every new match.

Search from the repository root:

```sh
rg -n 'CanonicalizeExample|enumvalue\.Normalize|protoJSONExample|normalizeOpenAPIExample|PayloadEx|StreamingPayloadEx|\.Example\(' expr codegen http/codegen jsonrpc/codegen grpc/codegen internal --glob '*.go' --glob '!**/*_test.go' --glob '!**/testdata/**' --glob '!**/gen/**'
rg -n 'jsonExample|flagDefault|NewFlagData|Example:|DefaultValue:' codegen/cli http/codegen/client_cli.go grpc/codegen/client_cli.go
```

Each row classifies a production file returned by the first search. Multiple
matches inside one file inherit its disposition; implementation records each
migrated call or retained compatibility boundary. The second search records
formatters/data carriers that do not invoke `.Example` themselves.

| Existing file | Disposition |
| --- | --- |
| `expr/example.go` | Semantic source: authored precedence, suppression, synthesis outcome |
| `expr/example_length.go` | Synthesis internals: preserve size constraints and branch information |
| `expr/types.go` | Semantic source: scalar, collection and union generation |
| `expr/types_union.go` | Union generation/selection: preserve selected occurrence identity |
| `expr/user_type.go` | Source identity and recursive memoization; no wire-value cache |
| `expr/example_canonicalization.go` | Replace duplicate selection/coercion with resolver; retain public JSON-shape adapter |
| `expr/json_schema_inline.go` | Migrate examples/defaults/enums to shared JSON value rules |
| `expr/http_body_types.go` | Transport-derived expression examples: retain authored provenance |
| `internal/enumvalue/value.go` | Retire semantic duplication after enum/default consumers migrate |
| `codegen/validation_render.go` | Consume resolved enum semantics without changing valid runtime acceptance |
| `codegen/service/service_data.go` | Data carriers: preserve semantic provenance across method/stream records |
| `codegen/service/service_data_methods.go` | Source selection for payload, result, stream and error data |
| `http/codegen/service_data_payload.go` | JSON body examples/defaults migrate; distinguish raw byte/text flags |
| `http/codegen/service_data_init_args.go` | Plain auth/header/cookie flag codecs retained; source examples use common precedence |
| `http/codegen/service_data_routes.go` | Path examples retain route codec; no JSON-base64 substitution |
| `http/codegen/service_data_body_types.go` | Derived body expressions retain source identity |
| `http/codegen/service_data_transport_helpers.go` | Location-specific examples retain codec; reconcile scalar rules |
| `http/codegen/service_data_type_registry.go` | Type-analysis example data: source identity, no independent synthesis |
| `http/codegen/websocket.go` | Streaming data: carry resolved source; retain framing and runtime |
| `http/codegen/client_cli.go` | Flag routing and optional example outcome; body vs plain codec distinction |
| `http/codegen/misc_sections.go` | Rendering consumer; no semantic decisions |
| `http/codegen/openapi/internal/ir/analyzer.go` | Shared examples/defaults/enums, visibility and final schema checks |
| `http/codegen/openapi/internal/ir/document.go` | Authored/synthesized examples, suppression and diagnostics |
| `http/codegen/openapi/internal/ir/document_examples.go` | Shared JSON projection; remove coercion/reselection, retain visibility rules |
| `http/codegen/openapi/internal/ir/example_generation.go` | Delete private graph-copy synthesis after common source migration |
| `http/codegen/openapi/internal/ir/document_cookies.go` | Preserve cookie transport codec and correct schema example representation |
| `http/codegen/openapi/v3/example.go` | Versioned rendering only; retain structured/external example metadata |
| `grpc/codegen/service_data_analysis.go` | Preserve authored request-message source through transport copies |
| `grpc/codegen/service_data_helpers.go` | Attribute analysis source identity; no replacement synthesis |
| `grpc/codegen/service_data_convert.go` | Converted message examples retain source/projection identity |
| `grpc/codegen/client_cli_example.go` | Replace resynthesis with protobuf-only projection of resolved values |

Additional boundaries: `expr/random.go` memo ownership; `expr/attribute.go`
example inheritance; `expr/attribute_validate.go` invalid authored values;
`internal/examplevalue/union.go` existing temporary selection representation;
`internal/jsonkey/key.go` shared member spelling; `codegen/cli/cli.go`,
`payload.go`, `conversion.go`, `command_data.go` flag data/formatting/defaults;
`grpc/codegen/client_cli.go` protojson routing. `codegen/cli/payload.go` owns `flagDefault`; `flag_parsing.go` renders the flag
default value. `codegen/cli/usage.go` renders per-command and aggregate help;
milestone 5 must propagate omission through both paths, with
`codegen/cli/usage_test.go` and generated CLI tests. Test-only matches from the
second search are regression anchors, not production consumers; retain and
extend their assertions when their corresponding formatter changes.

JSON-RPC reuses HTTP service analysis and CLI generation. Its lack of direct
matches does not exempt it: compile and exercise the JSON-RPC output, including
body defaults, errors and streaming-message surfaces. Generated service/client
interfaces, body DTO pointer/presence wrappers, protobuf oneof wrappers, example
service implementations and custom projector test doubles are compile-impact
surfaces even when their public shape must stay unchanged.

## #570 ownership checkpoint

The inventory searches were rerun for the #570 candidate against parent
`cd6d2fb03afa2381d42819a9bd50e364b419b3c6`. Service method payloads, results,
errors and streaming values now enter through `codegen/service/value_data.go`:
one `ValueContext` selects the source, resolves or synthesizes it once, and
publishes both the retained semantic result and its legacy raw example.
Package reanalysis reuses that same result. HTTP transport carriers preserve the
service source and add an occurrence-specific representation plan; their legacy
rendering consumers are still assigned to the later milestones below.

`expr/value_synthesis.go` is the new inventory match. Its calls into the existing
`AttributeExpr.Example` sampler occur inside an isolated synthesis graph; its
union adapter retains occurrence and branch identities before resolution. The
existing scalar/length/collection sampling helpers and recursion memo remain the
sampling engine, not a second transport projection owner. Public raw-value and
canonicalization APIs remain compatibility paths during the staged migration.

The remaining inline-schema/enum/default matches belong to #571, OpenAPI matches
to #572, HTTP/JSON-RPC/CLI matches to #565, and protobuf matches to #434. #573
must rerun the inventory and retire duplicate built-in interpretation only after
those consumers migrate. Adding carriers does not mean their old rendering
paths have already migrated. `PayloadEx`/`StreamingPayloadEx` declarations and
`renderPayloadExtraction` are carrier/name matches, not extra source-selection
implementations. The formatter search retains the same location-codec and CLI
migration owners recorded above.

## #565 execution boundary

After #572, the immediate HTTP/JSON-RPC defect is the CLI body argument's
independent `CanonicalizeExample(body.Example(...))` path. #565 replaces that
advertised example with the retained result projected through the client
`TypeData.Value` runtime plan. Shared CLI formatting carries availability through
both JSON error constructors, sample invocations, individual and aggregate help,
and the top-level example heading. It does not change command availability.

The remaining `cliBodyDefault` canonicalization and `AttributeExpr.Example`
calls in plain/location, response, type-data and WebSocket carriers are retained
explicitly for #573's consumer reconciliation. Their existing codecs and default
behavior remain protected by #565's focused controls. This allocation does not
declare those consumers migrated or remove them from the migration goal.

## #571 effective-constraint checkpoint

The accepted alias policy is refinement. `expr.EffectiveConstraints` owns the
immutable enum, default, numeric/length, format/pattern, and finalized required
field result for one occurrence and its declaration ancestry. Raw attribute
queries enter through `EffectiveConstraintsFor`; consumers do not walk named
aliases or choose precedence independently. Enum subset and default membership
reuse declared-type value resolution and equality. `internal/enumvalue` remains
only a JSON-shape projection boundary after semantic selection.

Pattern and format predicates are ordered current-to-base and conjoined. The
immutable result retains typed provenance; its detached lowered carrier keeps
explicit per-kind clauses when HTTP removes named wrappers. Generated
validators, value plans, inline schemas, OpenAPI analysis, examples, synthesis,
and vet consume those clauses rather than the singular compatibility fields.
Enum candidates are a separate presence-aware filtered view; physical lowering
keeps the unfiltered enum clauses and does not turn them into authored values.

## Existing proof anchors

- `http/codegen/testdata/mapped_names_dsls.go`: authored bytes, mapped keys,
  byte/text enums and untagged request/result bodies.
- `http/codegen/testdata/type_identity_dsl.go`: nested union and shared-type
  identity coverage through `TypeIdentityDSL`.
- `grpc/codegen/testdata/dsls_15.go` and
  `grpc/codegen/client_cli_protojson_{test,corpus_test}.go`: protojson output,
  generated payload builders, nested oneofs and wrapped values.
- `http/codegen/collection_defaults_cli_test.go` and
  `jsonrpc/codegen/collection_defaults_cli_test.go`: generated default semantics.
- `http/codegen/openapi/internal/ir/{byte_values_test.go,document_test.go,nested_union_examples_test.go,untagged_byte_examples_test.go}`:
  Any/custom/null, branch identity and typed-byte regressions.
- `internal/enumvalue/value_test.go`: Any values and wire-name precedence.
- `internal/openapiimport/issue_302_test.go`: original untagged compatibility need.
- `internal/testdatacompile/README.md`: discovered design corpus and explicit
  expected-failure policy.

The temporary comparison in `/tmp/loom-goa-audit` is supporting discovery
material, not a durable test gate. Historical sources are pinned in the design;
implementation must turn applicable minimal cases into checked-in Loom tests.
