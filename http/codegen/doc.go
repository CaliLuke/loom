/*
Package codegen generates HTTP servers, clients, and OpenAPI specifications
from an evaluated Loom design.

# Pipeline position

	service.ServicesData              (codegen/service)
	        |
	        v
	httpcodegen.NewServicesData(svc, API.HTTP)
	        |
	        +--> per service: analyze(HTTPServiceExpr) -> *ServiceData
	        |       which embeds *service.Data and adds transport-specific
	        |       endpoint data (routes, params, headers, bodies, errors).
	        |
	        v
	Section renderers (server.go, client.go, server_types.go, client_types.go,
	paths.go, cli_sections.go, openapi.go)
	        |
	        v
	[]*codegen.File consumed by codegen/generator

# What this package owns

  - HTTP-level endpoint shape: routes, path/query/header/cookie bindings,
    request/response bodies, status codes, error group mapping to status codes
    ([EndpointData], [RequestData], [ResponseData], [ErrorGroupData]).
  - Body element types ([ParamData], [HeaderData], [CookieData], [TypeData],
    [MultipartData]) — see service_data_element_types.go.
  - Transport-facing init code generation ([InitData] builders in
    service_data_payload.go, service_data_response.go,
    service_data_response_errors.go, service_data_init_args.go).
  - Server/client encode/decode and validation emission.
  - OpenAPI v3 generation (openapi/ sub-package) plus the internal IR
    (openapi/internal/ir) that decouples the design from the spec shape.
  - WebSocket/SSE templating and fixture generation for streaming endpoints.

# Intermediate representation

Endpoint wire shape flows through http/codegen/internal/transportir before the
final ServiceData is produced. transportir decouples "what the endpoint looks
like on the wire" (routes, parameters, bodies, responses) from the Go types
emitted for it, which keeps the response/payload/body builders small.

# File layout

  - service_data.go                     — ServicesData/ServiceData/EndpointData + entry points.
  - service_data_element_types.go       — Route/Param/Header/Cookie/Type/Multipart types.
  - service_data_payload.go             — payloadBuilder: request side.
  - service_data_response.go            — resultBuilder: success responses.
  - service_data_response_errors.go     — errorBuilder: error responses + Problem Details.
  - service_data_init_args.go           — shared InitArg constructors.
  - service_data_analysis.go            — analyze() entry point.
  - service_data_body_types.go          — request/response body type emission.
  - service_data_multipart.go           — multipart encode/decode data.
  - server.go / client.go / paths.go    — section renderers.
  - server_types.go / client_types.go   — type-file renderers.
  - cli_sections.go / example_*.go      — client CLI + example emission.
  - openapi.go + openapi/               — OpenAPI spec emission.
  - template_sources*.go / funcs.go     — string templates used by renderers.
  - websocket*.go / sse_client.go       — streaming codegen.
  - internal/transportir                — wire-shape IR.

# Extension points

  - Add a new HTTP element kind (e.g., trailer): extend
    service_data_element_types.go and add a builder in service_data_payload.go
    or service_data_response.go.
  - Add a new response variant (e.g., a second Problem profile): extend
    errorBuilder in service_data_response_errors.go.
  - Add a new template section: drop a new file alongside server_sections.go
    or client_sections.go and register it in server.go / client.go.

# Invariants

  - Transport types NEVER reach into service.Data's unexported fields;
    service-level state is accessed via its public methods or the already-
    populated embedded *service.Data.
  - Body element type names are registered through registerRequestBodyTypeNames
    and registerClientBodyType — do NOT mutate ServerTypeNames/ClientTypeNames
    directly from new call sites.
  - Views propagate through MethodData.ViewedResult; builders check viewed once
    and branch, they do not re-derive view selection.
*/
package codegen
