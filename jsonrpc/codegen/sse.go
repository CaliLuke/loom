package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// SSEServerFiles returns the generated JSON-RPC SSE server files if any.
func SSEServerFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	var files []*codegen.File
	jsvcs := data.Root.API.JSONRPC.Services
	for _, svc := range jsvcs {
		if f := sseServerFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
		if f := sseClientFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// sseServerFile returns the file implementing the SSE server streaming implementation if any.
func sseServerFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if data == nil {
		return nil
	}

	// Check if any endpoint has SSE
	hasSSE := false
	for _, ed := range data.Endpoints {
		if ed.SSE != nil {
			hasSSE = true
			break
		}
	}
	if !hasSSE {
		return nil
	}

	path := filepath.Join(codegen.Gendir, "jsonrpc", codegen.SnakeCase(svc.Name()), "server", "stream.go")
	streamSections := sseServerStreamSections(data)
	sections := make([]codegen.Section, 0, 1+len(streamSections))
	sections = append(sections,
		codegen.Header(
			"stream",
			"server",
			[]*codegen.ImportSpec{
				{Path: "context"},
				{Path: "errors"},
				{Path: "fmt"},
				{Path: "io"},
				{Path: "net/http"},
				{Path: "sync"},
				codegen.LoomImport(""),
				codegen.LoomImport("jsonrpc"),
				codegen.LoomNamedImport("http", "loomhttp"),
				codegen.LoomNamedImport("observability/transport", "loomtransport"),
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
			},
		),
	)
	for _, section := range streamSections {
		sections = append(sections, section)
	}
	return &codegen.File{Path: path, Sections: sections}
}

// sseClientFile returns the file implementing the SSE client streaming implementation if any.
func sseClientFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if data == nil {
		return nil
	}

	// Check if any endpoint has SSE
	hasSSE := false
	for _, ed := range data.Endpoints {
		if ed.SSE != nil {
			hasSSE = true
			break
		}
	}
	if !hasSSE {
		return nil
	}

	path := filepath.Join(codegen.Gendir, "jsonrpc", codegen.SnakeCase(svc.Name()), "client", "stream.go")
	tmplSections := sseClientStreamSections(data)
	sections := make([]codegen.Section, 0, 1+len(tmplSections))
	sections = append(sections,
		codegen.Header(
			"stream",
			"client",
			[]*codegen.ImportSpec{
				{Path: "bufio"},
				{Path: "bytes"},
				{Path: "context"},
				{Path: "encoding/json/jsontext"},
				{Path: "encoding/json/v2", Name: "json"},
				{Path: "fmt"},
				{Path: "io"},
				{Path: "net/http"},
				{Path: "strings"},
				{Path: "sync"},
				codegen.LoomImport("jsonrpc"),
				codegen.LoomNamedImport("http", "loomhttp"),
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
			},
		),
	)
	for _, section := range tmplSections {
		sections = append(sections, section)
	}
	return &codegen.File{Path: path, Sections: sections}
}

// sseServerStreamSections returns sections for SSE server endpoints.
func sseServerStreamSections(data *httpcodegen.ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0)
	for _, ed := range data.Endpoints {
		if ed.SSE == nil {
			continue
		}
		sections = append(sections, jsonrpcSSEServerStreamSection(ed))
	}
	return sections
}

// sseClientStreamSections returns section templates for SSE client endpoints.
func sseClientStreamSections(data *httpcodegen.ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0)
	for _, ed := range data.Endpoints {
		if ed.SSE == nil {
			continue
		}
		sections = append(sections, jsonrpcSSEClientStreamSection(ed))
	}
	return sections
}
