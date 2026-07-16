package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
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
	f.Path = jsonrpcTransportPath(f.Path)
	return f
}

func clientEncodeDecodeSections(f *codegen.File, svcData *httpcodegen.ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0, len(f.AllSections())+len(svcData.Endpoints))
	for _, section := range f.AllSections() {
		switch section.SectionName() {
		case "source-header":
			addJSONRPCClientImports(section)
		case "response-decoder":
			ed, ok := endpointDataForSection(section)
			if !ok {
				continue
			}
			sections = append(sections, jsonrpcResponseDecoderSection(svcData.Endpoint(ed.Method.Name)))
			continue
		}
		if section.SectionName() != "source-header" {
			section = renameJSONRPCSection(section, "jsonrpc-"+section.SectionName())
		}
		sections = append(sections, section)
	}

	for _, endpoint := range svcData.Endpoints {
		if endpoint.RequestEncoder == "" {
			sections = append(sections, jsonrpcMinimalRequestEncoderSection(endpoint))
			endpoint.RequestEncoder = fmt.Sprintf("Encode%sRequest", endpoint.Method.VarName)
		}
	}

	return sections
}

func addJSONRPCClientImports(section codegen.Section) {
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "bufio"})
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "bytes"})
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "sync"})
	codegen.AddSectionImport(section, &codegen.ImportSpec{Path: "sync/atomic"})
	codegen.AddSectionImport(section, codegen.LoomImport("jsonrpc"))
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
			codegen.LoomImport(""),
			codegen.LoomImport("jsonrpc"),
			codegen.LoomNamedImport("http", "loomhttp"),
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
