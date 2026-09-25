package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func websocketClientFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if !httpcodegen.HasWebSocket(data) {
		return nil
	}

	svcName := data.Service.PathName
	title := fmt.Sprintf("%s WebSocket JSON-RPC client", svc.Name())

	// Build imports list for WebSocket clients. The loom package declares
	// loom.JSONValue, the Go type of Any results and payloads.
	imports := make([]*codegen.ImportSpec, 0, 9+len(data.Service.UserTypeImports))
	imports = append(imports,
		&codegen.ImportSpec{Path: "bytes"},
		&codegen.ImportSpec{Path: "context"},
		&codegen.ImportSpec{Path: "fmt"},
		&codegen.ImportSpec{Path: "io"},
		&codegen.ImportSpec{Path: "net/http"},
		codegen.LoomImport("jsonrpc"),
		codegen.LoomNamedImport("http", "loomhttp"),
		codegen.LoomImport(""),
		&codegen.ImportSpec{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
	)
	imports = append(imports, data.Service.UserTypeImports...)
	data, imports = services.FileData(svc.Name(), imports)

	sections := []codegen.Section{
		codegen.Header(title, "client", imports),
	}

	sections = append(sections, jsonrpcWebSocketStreamErrorTypesSection())

	// Process only WebSocket endpoints and generate stream implementations only
	for _, e := range data.Endpoints {
		if !httpcodegen.IsWebSocketEndpoint(e) {
			continue
		}

		sections = append(sections, jsonrpcWebSocketClientStreamSection(e.ClientWebSocket))
	}

	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, "jsonrpc", svcName, "client", "websocket.go"),
		Sections: sections,
	}
}

// allErrors returns all errors for the given service.
func allErrors(data *httpcodegen.ServiceData) []*httpcodegen.ErrorData {
	seen := make(map[string]struct{})
	var errors []*httpcodegen.ErrorData
	for _, e := range data.Endpoints {
		for _, gerr := range e.Errors {
			for _, err := range gerr.Errors {
				if _, ok := seen[err.Name]; ok {
					continue
				}
				seen[err.Name] = struct{}{}
				errors = append(errors, err)
			}
		}
	}
	return errors
}
