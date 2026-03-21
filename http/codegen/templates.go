package codegen

import (
	"embed"

	"goa.design/goa/v3/codegen/template"
)

// Template constants
const (
	// Server templates
	serverStructT      = "server_struct"
	serverInitT        = "server_init"
	serverServiceT     = "server_service"
	serverUseT         = "server_use"
	serverMethodNamesT = "server_method_names"
	serverMountT       = "server_mount"
	serverHandlerT     = "server_handler"
	serverHandlerInitT = "server_handler_init"

	// Client templates
	clientStructT       = "client_struct"
	clientInitT         = "client_init"
	clientEndpointInitT = "client_endpoint_init"

	// Common templates
	transformHelperT = "transform_helper"
	pathT            = "path"
	pathInitT        = "path_init"
	requestInitT     = "request_init"

	// Endpoint templates
	parseEndpointT  = "parse_endpoint"
	requestBuilderT = "request_builder"

	// Encoder/Decoder templates
	requestEncoderT  = "request_encoder"
	responseDecoderT = "response_decoder"
	responseEncoderT = "response_encoder"
	requestDecoderT  = "request_decoder"
	errorEncoderT    = "error_encoder"

	// Multipart templates
	multipartRequestEncoderT     = "multipart_request_encoder"
	multipartRequestDecoderT     = "multipart_request_decoder"
	multipartRequestDecoderTypeT = "multipart_request_decoder_type"

	// WebSocket templates

	// SSE templates
	serverSseT = "server_sse"

	// File server templates
	appendFsT   = "append_fs"
	fileServerT = "file_server"

	// Mount point templates
	mountPointStructT = "mount_point_struct"

	// Stream templates
	// Partial templates
	sseFormatP              = "sse_format"
	sseParseP               = "sse_parse"
	clientTypeConversionP   = "client_type_conversion"
	clientMapConversionP    = "client_map_conversion"
	singleResponseP         = "single_response"
	queryTypeConversionP    = "query_type_conversion"
	elementSliceConversionP = "element_slice_conversion"
	sliceItemConversionP    = "slice_item_conversion"
	querySliceConversionP   = "query_slice_conversion"
	responseP               = "response"
	headerConversionP       = "header_conversion"
	requestElementsP        = "request_elements"
	queryMapConversionP     = "query_map_conversion"
	pathConversionP         = "path_conversion"
)

//go:embed templates/*
var templateFS embed.FS

// httpTemplates is the shared template reader for the http codegen package.
var httpTemplates = &template.TemplateReader{FS: templateFS}
