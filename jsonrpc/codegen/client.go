package codegen

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	httpcodegen "goa.design/goa/v3/http/codegen"
)

// ClientFiles returns the generated HTTP client files.
func ClientFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	jsvcs := data.Root.API.JSONRPC.Services
	files := make([]*codegen.File, 0, len(jsvcs)*3)
	for _, svc := range jsvcs {
		files = append(files, clientFile(genpkg, svc, data))
		if f := websocketClientFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
		if f := sseClientFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	for _, svc := range jsvcs {
		svcData := data.Get(svc.Name())
		if f := clientEncodeDecodeFile(genpkg, svc, data, svcData); f != nil {
			files = append(files, f)
		}
	}
	return files
}

func clientEncodeDecodeFile(genpkg string, svc *expr.HTTPServiceExpr, data *httpcodegen.ServicesData, svcData *httpcodegen.ServiceData) *codegen.File {
	f := httpcodegen.ClientEncodeDecodeFile(genpkg, svc, data)
	if f == nil {
		return nil
	}
	updateHeader(f)
	f.SetSections(clientEncodeDecodeSections(f, svcData))
	f.Path = strings.Replace(f.Path, "/http/", "/jsonrpc/", 1)
	return f
}

func clientEncodeDecodeSections(f *codegen.File, svcData *httpcodegen.ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0, len(f.AllSections())+len(svcData.Endpoints))
	for _, section := range f.AllSections() {
		s, ok := section.(*codegen.SectionTemplate)
		if !ok {
			sections = append(sections, section)
			continue
		}
		switch s.Name {
		case "source-header":
			addJSONRPCClientImports(s)
		case "request-encoder":
			rewriteJSONRPCRequestEncoder(s)
		case "response-decoder":
			ed, ok := s.Data.(*httpcodegen.EndpointData)
			if !ok {
				continue
			}
			sections = append(sections, jsonrpcResponseDecoderSection(svcData.Endpoint(ed.Method.Name)))
			continue
		}
		if s.Name != "source-header" {
			s.Name = "jsonrpc-" + s.Name
		}
		sections = append(sections, s)
	}

	for _, endpoint := range svcData.Endpoints {
		if endpoint.RequestEncoder == "" {
			sections = append(sections, jsonrpcMinimalRequestEncoderSection(endpoint))
			endpoint.RequestEncoder = fmt.Sprintf("Encode%sRequest", endpoint.Method.VarName)
		}
	}

	return sections
}

func addJSONRPCClientImports(section *codegen.SectionTemplate) {
	codegen.AddImport(section, &codegen.ImportSpec{Path: "bufio"})
	codegen.AddImport(section, &codegen.ImportSpec{Path: "bytes"})
	codegen.AddImport(section, &codegen.ImportSpec{Path: "sync"})
	codegen.AddImport(section, &codegen.ImportSpec{Path: "sync/atomic"})
	codegen.AddImport(section, &codegen.ImportSpec{Path: "github.com/google/uuid"})
	codegen.AddImport(section, codegen.GoaImport("jsonrpc"))
}

func rewriteJSONRPCRequestEncoder(section *codegen.SectionTemplate) {
	re := regexp.MustCompile(`body := (.*)\n`)
	section.Source = re.ReplaceAllStringFunc(section.Source, func(match string) string {
		matches := re.FindStringSubmatch(match)
		return strings.Replace(newJSONRPCBody, "{{ .NewBody }}", matches[1], 1)
	})
}

// clientFile returns the client HTTP transport file
func clientFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "jsonrpc", svcName, "client", "client.go")
	title := fmt.Sprintf("%s client JSON-RPC transport", svc.Name())
	sections := []codegen.Section{
		codegen.Header(title, "client", []*codegen.ImportSpec{
			{Path: "bufio"},
			{Path: "bytes"},
			{Path: "context"},
			{Path: "fmt"},
			{Path: "io"},
			{Path: "net/http"},
			{Path: "strconv"},
			{Path: "strings"},
			{Path: "sync"},
			{Path: "sync/atomic"},
			{Path: "time"},
			{Path: "github.com/gorilla/websocket"},
			codegen.GoaImport(""),
			codegen.GoaImport("jsonrpc"),
			codegen.GoaNamedImport("http", "goahttp"),
			{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
			{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
		}),
	}
	sections = append(sections, jsonrpcClientStructSection(data))
	sections = append(sections, jsonrpcClientInitSection(data))

	for _, e := range data.Endpoints {
		sections = append(sections, jsonrpcClientEndpointInitSection(e))
	}

	if httpcodegen.HasWebSocket(data) {
		sections = append(sections, jsonrpcWebSocketClientConnSection(data))
	}

	return &codegen.File{Path: path, Sections: sections}
}

const newJSONRPCBody = `b := {{ .NewBody }}
		body := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "{{ .Method.Name }}",
			Params:  b,
		}
{{- if .Payload.IDAttribute }}
	{{- if .Payload.IDAttributeRequired }}
		if p.{{ .Payload.IDAttribute }} != "" {
			body.ID = p.{{ .Payload.IDAttribute }}
		}
		// If ID is empty, this is a notification - no ID field
	{{- else }}
		if p.{{ .Payload.IDAttribute }} != nil && *p.{{ .Payload.IDAttribute }} != "" {
			body.ID = p.{{ .Payload.IDAttribute }}
		}
		// If ID is nil or empty, this is a notification - no ID field
	{{- end }}
{{- else }}
		// No ID field in payload - always send as a request with generated ID
		id := uuid.New().String()
		body.ID = id
{{- end }}
`
