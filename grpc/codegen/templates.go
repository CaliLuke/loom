package codegen

import (
	"embed"

	"github.com/CaliLuke/loom/codegen/template"
)

// Proto template constants
const (
	grpcProtoHeaderT = "proto_header"
	grpcProtoStartT  = "proto_start"
	grpcServiceT     = "grpc_service"
	grpcMessageT     = "grpc_message"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// grpcTemplates is the shared template reader for the grpc codegen package (package-private).
var grpcTemplates = &template.TemplateReader{FS: templateFS, Extension: ".tmpl"}
