package service

import (
	"embed"

	"goa.design/goa/v3/codegen/template"
)

// Template constants
const (
	// Endpoint templates
	serviceEndpointMethodT = "service_endpoint_method"

	// Service templates
	serviceT = "service"

	// Interceptor templates
	interceptorsT                        = "interceptors"
	interceptorsTypesT                   = "interceptors_types"
	serverInterceptorsT                  = "server_interceptors"
	clientInterceptorsT                  = "client_interceptors"
	endpointWrappersT                    = "endpoint_wrappers"
	clientWrappersT                      = "client_wrappers"
	serverInterceptorStreamWrapperTypesT = "server_interceptor_stream_wrapper_types"
	clientInterceptorStreamWrapperTypesT = "client_interceptor_stream_wrapper_types"
	serverInterceptorWrappersT           = "server_interceptor_wrappers"
	clientInterceptorWrappersT           = "client_interceptor_wrappers"
	serverInterceptorStreamWrappersT     = "server_interceptor_stream_wrappers"
	clientInterceptorStreamWrappersT     = "client_interceptor_stream_wrappers"
)

//go:embed templates/*
var templateFS embed.FS

// serviceTemplates is the shared template reader for the service codegen package (package-private).
var serviceTemplates = &template.TemplateReader{FS: templateFS}
