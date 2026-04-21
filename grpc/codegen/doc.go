/*
Package codegen generates gRPC servers, clients, and .proto definitions from an
evaluated Loom design. It bridges the transport-neutral data produced by
codegen/service with the protoc/protoc-gen-go toolchain.

# Pipeline position

	service.ServicesData             (codegen/service)
	        |
	        v
	grpccodegen.NewServicesData(svc)
	        |
	        +--> per service: analyze(GRPCServiceExpr) -> *ServiceData
	        |       which embeds *service.Data and adds:
	        |         - Proto message shape + imports
	        |         - Metadata bindings (headers, trailers)
	        |         - gRPC status code mapping for errors
	        |         - Server/client stream interfaces
	        |
	        v
	Section renderers (proto.go, server.go, client.go, server_types.go,
	client_types.go, codec_sections.go, client_cli.go)
	        |
	        v
	[]*codegen.File consumed by codegen/generator
	        |
	        v
	protoc + protoc-gen-go + protoc-gen-go-grpc  (invoked by loom gen)

# What this package owns

  - Proto message synthesis: user-type → protobuf type mapping, including
    nested types, unions (as `oneof`), and repeated/map handling
    (protobuf.go, protobuf_transform*.go).
  - Proto IR for endpoints: request/response messages, RPC method declaration,
    metadata attachments (service_data_messages.go).
  - Transport codec generation: converting between Loom payload/result types
    and protoc-generated Go types (service_data_convert.go, codec_sections.go).
  - gRPC-specific validation emission (service_data_validation.go).
  - Streaming server/client interfaces and their adaptation to loom
    streaming semantics (service_data_stream.go).
  - Example server/client and client CLI (example_*.go, client_cli.go).

# Key invariants

  - Proto names MUST be stable: field numbers, enum values, and message names
    are driven by the DSL. Do not derive proto names from Go identifiers
    produced elsewhere — use the dedicated proto name helpers.
  - Transforms route through NameScope helpers (GoTypeRef / GoFullTypeRef /
    GoTypeName); never concatenate strings to form type references
    (see CLAUDE.md rule 51).
  - Metadata bindings live on the transport side only. The base MethodData
    already knows about the payload/result types; this package adds the
    metadata mapping without mutating service.Data.

# File layout

  - service_data.go               — ServicesData/ServiceData + entry points.
  - service_data_analysis.go      — analyze() implementation.
  - service_data_messages.go      — proto message/request/response assembly.
  - service_data_convert.go       — Loom <-> proto type conversion metadata.
  - service_data_stream.go        — server/client streaming interfaces.
  - service_data_validation.go    — per-endpoint validation code.
  - service_data_helpers.go       — shared analyze helpers.
  - proto.go / protobuf*.go       — .proto rendering + transform building.
  - server.go / client.go         — server/client section renderers.
  - server_types.go / client_types.go — type-file renderers.
  - codec_sections.go             — marshal/unmarshal codecs.
  - templates, templates.go       — string templates used by renderers.

# Extension points

  - Add a new proto field-level option (e.g., a custom annotation): extend
    protobuf.go and plumb through the message builder in
    service_data_messages.go.
  - Add a new metadata binding source: extend service_data_messages.go and
    the codec emitter in codec_sections.go.
  - Add a new streaming mode: extend service_data_stream.go and the
    corresponding server/client renderers.
*/
package codegen
