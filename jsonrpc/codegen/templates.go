package codegen

import (
	"embed"
	"strings"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/template"
)

// Server template constants
const (
	// Server
	serverHandlerT     = "server_handler"
	serverHandlerInitT = "server_handler_init"

	// Client
	responseDecoderT = "response_decoder"

	// WebSocket templates
	websocketServerHandlerT = "websocket_server_handler"

	// JSON-RPC WebSocket client templates
	websocketClientStreamT = "websocket_client_stream"

	// SSE templates
	sseClientStreamT  = "sse_client_stream"
	sseServerHandlerT = "sse_server_handler"

	// Partial templates
	singleResponseP         = "single_response"
	queryTypeConversionP    = "query_type_conversion"
	elementSliceConversionP = "element_slice_conversion"
	sliceItemConversionP    = "slice_item_conversion"
)

//go:embed templates/*
var templateFS embed.FS

// jsonrpcTemplates is the shared template reader for the jsonrpc codegen package (package-private).
var jsonrpcTemplates = &template.TemplateReader{FS: templateFS}

// updateHeader modifies the header of the given file to be JSON-RPC specific.
func updateHeader(f *codegen.File) {
	// Update the title
	header := f.HeaderTemplate()
	if header == nil {
		return
	}
	data := codegen.HeaderSectionData(header)
	if data == nil {
		return
	}
	data.Title = strings.Replace(data.Title, "HTTP", "JSON-RPC", 1)

	// Update the imports
	for _, i := range data.Imports {
		i.Path = strings.Replace(i.Path, "gen/http", "gen/jsonrpc", 1)
	}
}
