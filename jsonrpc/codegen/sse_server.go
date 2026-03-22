package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/expr"
	httpcodegen "github.com/CaliLuke/loom/v3/http/codegen"
)

// sseServerStreamFile returns the file implementing the JSON-RPC SSE server
// streaming implementation if any.
func sseServerStreamFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if data == nil {
		return nil
	}

	// Check if service has streaming methods
	hasStreaming := false
	for _, m := range data.Service.Methods {
		if m.ServerStream != nil {
			hasStreaming = true
			break
		}
	}
	if !hasStreaming {
		return nil
	}

	svcName := data.Service.PathName
	title := fmt.Sprintf("%s SSE server streaming", svc.Name())
	imports := make([]*codegen.ImportSpec, 0, 11+len(data.Service.UserTypeImports))
	imports = append(imports,
		&codegen.ImportSpec{Path: "context"},
		&codegen.ImportSpec{Path: "errors"},
		&codegen.ImportSpec{Path: "fmt"},
		&codegen.ImportSpec{Path: "net/http"},
		&codegen.ImportSpec{Path: "sync"},
		codegen.GoaImport(""),
		codegen.GoaImport("jsonrpc"),
		codegen.GoaNamedImport("http", "goahttp"),
		// Import the service package from the correct location
		&codegen.ImportSpec{Path: genpkg + "/" + codegen.SnakeCase(data.Service.Name), Name: data.Service.PkgName},
	)
	imports = append(imports, data.Service.UserTypeImports...)
	sections := []codegen.Section{
		codegen.Header(title, "server", imports),
		jsonrpcSSEServerImplSection(data),
	}

	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, "jsonrpc", svcName, "server", "sse.go"),
		Sections: sections,
	}
}
