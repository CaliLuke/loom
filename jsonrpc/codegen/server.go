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
	f.Path = strings.Replace(f.Path, "/http/", "/jsonrpc/", 1)
	return f
}

func serverEncodeDecodeSections(f *codegen.File) []codegen.Section {
	sections := make([]codegen.Section, 0, len(f.AllSections()))
	for _, section := range f.AllSections() {
		s, ok := section.(*codegen.SectionTemplate)
		if !ok {
			sections = append(sections, section)
			continue
		}
		switch s.Name {
		case "source-header":
			addJSONRPCServerImports(s)
		case "request-decoder":
			rewriteJSONRPCRequestDecoder(s)
			s.Name = "jsonrpc-request-decoder"
			sections = append(sections, s)
			continue
		case "error-encoder":
			continue
		}
		if s.Name != "source-header" {
			s.Name = "jsonrpc-" + s.Name
		}
		sections = append(sections, s)
	}
	return sections
}

func addJSONRPCServerImports(section *codegen.SectionTemplate) {
	codegen.AddImport(section, &codegen.ImportSpec{Path: "bytes"})
	codegen.AddImport(section, &codegen.ImportSpec{Path: "io"})
	codegen.AddImport(section, codegen.LoomImport("jsonrpc"))
}

func rewriteJSONRPCRequestDecoder(section *codegen.SectionTemplate) {
	section.Source = strings.Replace(section.Source,
		"func(*http.Request) (",
		"func(*http.Request, *jsonrpc.RawRequest) (", 1)

	section.Source = strings.Replace(section.Source,
		"return func(r *http.Request) ({{ .Payload.Ref }}, error) {",
		`return func(r *http.Request, req *jsonrpc.RawRequest) ({{ .Payload.Ref }}, error) {
		r.Body = io.NopCloser(bytes.NewReader(req.Params))`, 1)

	section.Source = strings.ReplaceAll(section.Source,
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
	imports := make([]*codegen.ImportSpec, 0, 15+len(data.Service.UserTypeImports))
	imports = append(imports,
		&codegen.ImportSpec{Path: "bufio"},
		&codegen.ImportSpec{Path: "bytes"},
		&codegen.ImportSpec{Path: "context"},
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
		&codegen.ImportSpec{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		&codegen.ImportSpec{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	)
	imports = append(imports, data.Service.UserTypeImports...)
	sections := []codegen.Section{
		codegen.Header(title, "server", imports),
	}

	sections = append(sections,
		jsonrpcServerStructSection(data),
		jsonrpcServerInitSection(data, hasJSONRPCSSE(svc, services), hasMixedJSONRPCTransports(svc, services)),
		jsonrpcServerServiceSection(data),
		jsonrpcServerUseSection(data),
		jsonrpcServerMethodNamesSection(data),
		jsonrpcServerResponseCaptureSection(),
	)

	// Use appropriate server handler based on transport
	switch {
	case hasMixedJSONRPCTransports(svc, services):
		// For mixed transports, we need a unified handler with content negotiation
		sections = append(sections, jsonrpcMixedServerHandlerSection(data))
		// Include the standard HTTP handlers that the mixed handler delegates to
		sections = append(sections, jsonrpcServerHandlerSection(data, true))
		// Also include SSE handler for SSE-specific logic
		sections = append(sections, jsonrpcSSEServerHandlerSection(data))
	case hasJSONRPCSSE(svc, services):
		sections = append(sections, jsonrpcSSEServerHandlerSection(data))
	case httpcodegen.HasWebSocket(data):
		sections = append(sections, jsonrpcWebSocketServerHandlerSection(data))
	default:
		sections = append(sections, jsonrpcServerHandlerSection(data, false))
	}

	// Add transport flags to data
	mountData := struct {
		*httpcodegen.ServiceData
		HasSSE   bool
		HasMixed bool
	}{
		ServiceData: data,
		HasSSE:      hasJSONRPCSSE(svc, services),
		HasMixed:    hasMixedJSONRPCTransports(svc, services),
	}

	sections = append(sections,
		jsonrpcServerMountSection(data, mountData.HasSSE, mountData.HasMixed),
	)

	for _, e := range data.Endpoints {
		sections = append(sections, jsonrpcServerHandlerInitSection(e))
	}

	if !httpcodegen.HasWebSocket(data) {
		sections = append(sections, jsonrpcServerEncodeErrorSection(data.ServerStruct))
	}

	return &codegen.File{Path: fpath, Sections: sections}
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
