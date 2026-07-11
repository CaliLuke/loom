package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ServerFiles returns the generated JSON-RPC server files if any.
func ServerFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	jsvcs := data.Root.API.JSONRPC.Services
	files := make([]*codegen.File, 0, len(jsvcs)*3)
	for _, svc := range jsvcs {
		files = append(files, serverFile(genpkg, svc, data))
		// Generate either WebSocket or SSE file based on transport type
		if hasJSONRPCSSE(svc, data) {
			if f := sseServerStreamFile(genpkg, svc, data); f != nil {
				files = append(files, f)
			}
		} else if f := websocketServerFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	for _, svc := range jsvcs {
		if f := serverEncodeDecodeFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	return files
}

func serverEncodeDecodeFile(genpkg string, svc *expr.HTTPServiceExpr, data *httpcodegen.ServicesData) *codegen.File {
	f := httpcodegen.ServerEncodeDecodeFile(genpkg, svc, data)
	if f == nil {
		return nil
	}
	updateHeader(f)
	f.SetSections(serverEncodeDecodeSections(f))
	f.Path = rewriteJSONRPCTransportPath(f.Path)
	return f
}

func serverEncodeDecodeSections(f *codegen.File) []codegen.Section {
	sections := make([]codegen.Section, 0, len(f.AllSections()))
	for _, section := range f.AllSections() {
		switch section.SectionName() {
		case "source-header":
			addJSONRPCServerImports(section)
		case "request-decoder":
			section = rewriteJSONRPCSectionSource(section, rewriteJSONRPCRequestDecoderSource)
			sections = append(sections, renameJSONRPCSection(section, "jsonrpc-request-decoder"))
			continue
		case "error-encoder":
			continue
		}
		if section.SectionName() != "source-header" {
			section = renameJSONRPCSection(section, "jsonrpc-"+section.SectionName())
		}
		sections = append(sections, section)
	}
	return sections
}

func addJSONRPCServerImports(section codegen.Section) {
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "bytes"})
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "io"})
	codegen.AddSectionImport(section, codegen.LoomImport("jsonrpc"))
}

func rewriteJSONRPCRequestDecoderSource(source string) string {
	source = strings.Replace(source,
		"func(*http.Request) (",
		"func(*http.Request, *jsonrpc.RawRequest) (", 1)

	source = strings.Replace(source,
		"return func(r *http.Request) ({{ .Payload.Ref }}, error) {",
		`return func(r *http.Request, req *jsonrpc.RawRequest) ({{ .Payload.Ref }}, error) {
		params := req.Params
		if len(params) == 0 {
			params = []byte("{}")
		}
		r.Body = io.NopCloser(bytes.NewReader(params))`, 1)

	return strings.ReplaceAll(source,
		"return nil, ",
		`var zero {{ .Payload.Ref }}
		return zero, `)
}

// serverFile returns the file implementing the HTTP server.
func serverFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "jsonrpc", svcName, "server", "server.go")
	title := fmt.Sprintf("%s JSON-RPC server", svc.Name())
	imports := jsonrpcServerImports(genpkg, svcName, data)
	sections := []codegen.Section{
		codegen.Header(title, "server", imports),
	}

	hasSSE := hasJSONRPCSSE(svc, services)
	hasMixed := hasMixedJSONRPCTransports(svc, services)
	sections = append(sections, jsonrpcServerBaseSections(data, hasSSE, hasMixed)...)

	sections = append(sections, jsonrpcServerTransportSections(data, hasSSE, hasMixed)...)
	sections = append(sections, jsonrpcServerMountSection(data, hasSSE, hasMixed))

	for _, e := range data.Endpoints {
		sections = append(sections, jsonrpcServerHandlerInitSection(e))
	}

	if !httpcodegen.HasWebSocket(data) {
		sections = append(sections, jsonrpcServerEncodeErrorSection(data.ServerStruct))
	}

	return &codegen.File{Path: fpath, Sections: sections}
}

func jsonrpcServerImports(genpkg, svcName string, data *httpcodegen.ServiceData) []*codegen.ImportSpec {
	imports := make([]*codegen.ImportSpec, 0, 15+len(data.Service.UserTypeImports))
	imports = append(imports,
		&codegen.ImportSpec{Path: "bufio"},
		&codegen.ImportSpec{Path: "bytes"},
		&codegen.ImportSpec{Path: "context"},
		&codegen.ImportSpec{Path: "encoding/json"},
		&codegen.ImportSpec{Path: "errors"},
		&codegen.ImportSpec{Path: "fmt"},
		&codegen.ImportSpec{Path: "io"},
		&codegen.ImportSpec{Path: "mime/multipart"},
		&codegen.ImportSpec{Path: "net/http"},
		&codegen.ImportSpec{Path: "path"},
		&codegen.ImportSpec{Path: "strings"},
		codegen.LoomImport(""),
		codegen.LoomImport("jsonrpc"),
		codegen.LoomNamedImport("http", "loomhttp"),
		codegen.LoomNamedImport("observability/transport", "loomtransport"),
		&codegen.ImportSpec{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		&codegen.ImportSpec{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	)
	return append(imports, data.Service.UserTypeImports...)
}

func jsonrpcServerBaseSections(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) []codegen.Section {
	sections := []codegen.Section{
		jsonrpcServerStructSection(data),
		jsonrpcServerInitSection(data, hasSSE, hasMixed),
		jsonrpcServerServiceSection(data),
		jsonrpcServerUseSection(data),
		jsonrpcServerMethodNamesSection(data),
	}
	if serviceNeedsJSONRPCResponseCapture(data) {
		sections = append(sections, jsonrpcServerResponseCaptureSection())
	}
	return sections
}

func jsonrpcServerTransportSections(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) []codegen.Section {
	switch {
	case hasMixed:
		return []codegen.Section{
			jsonrpcMixedServerHandlerSection(data),
			jsonrpcServerHandlerSection(data, true),
			jsonrpcSSEServerHandlerSection(data),
		}
	case hasSSE:
		return []codegen.Section{jsonrpcSSEServerHandlerSection(data)}
	case httpcodegen.HasWebSocket(data):
		return []codegen.Section{jsonrpcWebSocketServerHandlerSection(data)}
	default:
		return []codegen.Section{jsonrpcServerHandlerSection(data, false)}
	}
}

// lowerInitial returns the string with the first letter in lowercase.
func lowerInitial(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

// hasJSONRPCSSE returns true if the service uses SSE for JSON-RPC streaming.
func hasJSONRPCSSE(svc *expr.HTTPServiceExpr, data *httpcodegen.ServicesData) bool {
	svcData := data.Get(svc.Name())
	if svcData == nil {
		return false
	}

	// Check if any JSON-RPC streaming endpoint uses SSE
	for _, e := range svc.HTTPEndpoints {
		if e.MethodExpr.IsStreaming() && e.IsJSONRPC() && e.SSE != nil {
			return true
		}
	}

	return false
}

// hasJSONRPCHTTP returns true if the service has non-streaming JSON-RPC endpoints.
func hasJSONRPCHTTP(svc *expr.HTTPServiceExpr) bool {
	for _, e := range svc.HTTPEndpoints {
		if e.IsJSONRPC() && !e.MethodExpr.IsStreaming() {
			return true
		}
	}
	return false
}

// hasMixedJSONRPCTransports returns true if the service has both HTTP and SSE JSON-RPC endpoints.
func hasMixedJSONRPCTransports(svc *expr.HTTPServiceExpr, data *httpcodegen.ServicesData) bool {
	return hasJSONRPCHTTP(svc) && hasJSONRPCSSE(svc, data)
}
