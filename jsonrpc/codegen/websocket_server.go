package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// websocketServerFile returns the file implementing the JSON-RPC WebSocket server
// streaming implementation if any. It follows the exact same pattern as the encode/decode
// files: get the HTTP file and modify it for JSON-RPC.
func websocketServerFile(genpkg string, svc *expr.HTTPServiceExpr, services *httpcodegen.ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if !httpcodegen.HasWebSocket(data) {
		return nil
	}
	svcName := data.Service.PathName
	title := fmt.Sprintf("%s WebSocket server streaming", svc.Name())
	imports := make([]*codegen.ImportSpec, 0, 14+len(data.Service.UserTypeImports))
	imports = append(imports,
		&codegen.ImportSpec{Path: "context"},
		&codegen.ImportSpec{Path: "encoding/json"},
		&codegen.ImportSpec{Path: "errors"},
		&codegen.ImportSpec{Path: "fmt"},
		&codegen.ImportSpec{Path: "io"},
		&codegen.ImportSpec{Path: "net/http"},
		&codegen.ImportSpec{Path: "strings"},
		&codegen.ImportSpec{Path: "sync"},
		&codegen.ImportSpec{Path: "time"},
		&codegen.ImportSpec{Path: "github.com/gorilla/websocket"},
		codegen.LoomImport(""),
		codegen.LoomImport("jsonrpc"),
		codegen.LoomNamedImport("http", "loomhttp"),
		&codegen.ImportSpec{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
	)
	imports = append(imports, data.Service.UserTypeImports...)
	sections := make([]codegen.Section, 0, 1+len(data.Endpoints))
	sections = append(sections, codegen.Header(title, "server", imports))
	sections = append(sections, jsonrpcWebSocketServerSections(data)...)

	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, "jsonrpc", svcName, "server", "websocket.go"),
		Sections: sections,
	}
}
