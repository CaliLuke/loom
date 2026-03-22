package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/expr"
)

// sseClientFile returns the file implementing the SSE client code for SSE endpoints if any.
// Relies on SSEData (ed.SSE) for all codegen needs.
func sseClientFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
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
	path := filepath.Join(codegen.Gendir, "http", codegen.SnakeCase(svc.Name()), "client", "sse.go")
	streamSections := sseClientSections(data)
	sections := make([]codegen.Section, 0, 1+len(streamSections))
	sections = append(sections,
		codegen.Header(
			"sse-client",
			"client",
			[]*codegen.ImportSpec{
				{Path: "bytes"},
				{Path: "context"},
				{Path: "errors"},
				{Path: "io"},
				{Path: "net/http"},
				{Path: "fmt"},
				{Path: "strconv"},
				{Path: "strings"},
				{Path: "sync"},
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()) + "/views", Name: data.Service.ViewsPkg},
				{Path: "github.com/CaliLuke/loom/v3/http", Name: "goahttp"},
			},
		),
	)
	for _, section := range streamSections {
		sections = append(sections, section)
	}
	return &codegen.File{Path: path, Sections: sections}
}
