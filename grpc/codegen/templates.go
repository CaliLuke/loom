package codegen

import (
	"embed"

	"github.com/CaliLuke/loom/v3/codegen/template"
)

// Client template constants
const (
	grpcClientStructT       = "client_struct"
	grpcClientInitT         = "client_init"
	grpcClientEndpointInitT = "client_endpoint_init"
)

// Server template constants
const (
	grpcServerStructTypeT    = "server_struct_type"
	grpcServerInitT          = "server_init"
	grpcServerGRPCInitT      = "server_grpc_init"
	grpcServerGRPCInterfaceT = "server_grpc_interface"
	grpcServerGRPCRegisterT  = "server_grpc_register"
	grpcServerGRPCStartT     = "server_grpc_start"
	grpcServerGRPCEndT       = "server_grpc_end"
	grpcHandlerInitT         = "grpc_handler_init"
)

// Stream template constants
const (
	grpcStreamStructTypeT = "stream_struct_type"
	grpcStreamSendT       = "stream_send"
	grpcStreamRecvT       = "stream_recv"
	grpcStreamCloseT      = "stream_close"
	grpcStreamSetViewT    = "stream_set_view"
)

// Proto template constants
const (
	grpcProtoHeaderT = "proto_header"
	grpcProtoStartT  = "proto_start"
	grpcServiceT     = "grpc_service"
	grpcMessageT     = "grpc_message"
)

// CLI template constants
const (
	grpcDoGRPCCLIT           = "do_grpc_cli"
	grpcParseEndpointT       = "parse_endpoint"
	grpcRemoteMethodBuilderT = "remote_method_builder"
)

// Common template constants
const (
	grpcTypeInitT        = "type_init"
	grpcValidateT        = "validate"
	grpcTransformHelperT = "transform_helper"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// grpcTemplates is the shared template reader for the grpc codegen package (package-private).
var grpcTemplates = &template.TemplateReader{FS: templateFS, Extension: ".tmpl"}
