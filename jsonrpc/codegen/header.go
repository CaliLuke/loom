package codegen

import (
	"path"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

// updateHeader updates the header of f, a file that the HTTP code generator
// builds for the JSON-RPC transport. It names JSON-RPC in the title and
// moves the imports of the packages of the generated HTTP tree of genpkg to
// the JSON-RPC tree, see jsonrpcImportPath.
func updateHeader(f *codegen.File, genpkg string) {
	data := updateTitle(f)
	if data == nil {
		return
	}
	for _, i := range data.Imports {
		i.Path = jsonrpcImportPath(genpkg, i.Path)
	}
}

// updateTitle names JSON-RPC in place of HTTP in the header title of f, a
// file that the HTTP code generator builds for the JSON-RPC transport. It
// returns the header data of f, nil if f has no header.
func updateTitle(f *codegen.File) *codegen.HeaderData {
	header := f.HeaderSection()
	if header == nil {
		return nil
	}
	data := codegen.HeaderDataForSection(header)
	if data == nil {
		return nil
	}
	data.Title = strings.Replace(data.Title, "HTTP", "JSON-RPC", 1)
	return data
}

// jsonrpcImportPath returns the import path of the JSON-RPC package that
// corresponds to importPath when importPath is a package of the generated
// HTTP tree genpkg/http, such as genpkg/http/calc/server. It returns any
// other import path, such as the service package genpkg/http_ of a service
// named http, unchanged.
func jsonrpcImportPath(genpkg, importPath string) string {
	rel, ok := strings.CutPrefix(importPath, path.Join(genpkg, "http")+"/")
	if !ok {
		return importPath
	}
	return path.Join(genpkg, "jsonrpc", rel)
}
