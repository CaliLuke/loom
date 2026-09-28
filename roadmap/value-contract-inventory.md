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
