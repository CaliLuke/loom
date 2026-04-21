/*
Package service is the base stage of the Loom codegen pipeline. It turns the
evaluated DSL ([expr.RootExpr]) into a transport-neutral data model
([ServicesData]) consumed by every transport codegen.

# Pipeline position

	DSL functions (github.com/CaliLuke/loom/dsl)
	        |
	        v
	eval.Run -> expr.RootExpr                     [eval package]
	        |
	        v
	service.NewServicesData(root) -> *ServicesData  (this package)
	        |
	        +--> http/codegen.NewServicesData(svc, HTTP)
	        +--> grpc/codegen.NewServicesData(svc)
	        +--> jsonrpc/codegen uses http/codegen with JSONRPC.HTTPExpr
	        |
	        v
	codegen/generator emits *codegen.File per transport

Each transport's ServicesData embeds *service.ServicesData, so anything computed
here is visible to every transport without duplication.

# What this package owns

  - Service-level shape: name, API info, package names, scopes.
  - Method-level shape: payload/result/streaming/transport/security descriptors
    ([MethodData] and its embedded *Data sub-structs).
  - User types, error init data, and union types.
  - View projections and projected/viewed-result type data
    ([ProjectedTypeData], [ViewedResultTypeData]).
  - Interceptor data.

Transports MUST NOT re-derive any of the above. They consume it via the embedded
*service.ServicesData.

# What this package does NOT own

  - HTTP/gRPC/JSON-RPC wire framing, status codes, routes, metadata, or
    streaming transport details — those live in the respective transport
    codegen packages.
  - File emission or templates — those live in codegen/generator and the
    transport codegen packages.

# Lazy evaluation

ServicesData computes a service's Data lazily on first [ServicesData.Get]. Once
computed it is cached in the ServicesData.Services map. Callers can rely on
referential stability: Get returns the same *Data pointer on repeat calls.

# File layout

  - service_data.go            — top-level types + ServicesData/Data/MethodData methods.
  - service_data_view_types.go — view-related data types (projected / viewed).
  - service_data_types.go      — user types, unions, and scope helpers.
  - service_data_methods.go    — per-method analysis (payload/result/streaming).
  - service_data_analysis.go   — ServicesData.analyze (Get's computation backend).
  - service_data_interceptors.go — interceptor data collection.
  - service_data_views.go / _init.go / _validation.go — projected type builders.
  - convert*.go, endpoint*.go, interceptor*.go, security*.go, example*.go
    — section renderers and helpers.

# Extension points

  - Add a new method-level field: put the declaration in one of the
    Method*Data sub-structs in service_data.go, populate it from
    service_data_methods.go or service_data_analysis.go.
  - Add a new user-type transformation: extend service_data_types.go.
  - Add a new transport: create a sibling `codegen` package that embeds
    *service.ServicesData and adds transport-specific data + section
    renderers. Do not reach into private fields — add accessors here instead.
*/
package service
