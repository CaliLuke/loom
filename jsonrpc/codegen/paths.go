package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// PathFiles returns the service path files.
func PathFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.PathFiles(genpkg, data)
	for _, f := range res {
		// Path type imports keep their service package; only transport paths move.
		updateTitle(f)
		f.Path = jsonrpcTransportPath(f.Path)
	}
	return res
}

func jsonrpcTransportPath(filePath string) string {
	httpRoot := filepath.Join(codegen.Gendir, "http")
	rel, err := filepath.Rel(httpRoot, filePath)
	if err != nil || !filepath.IsLocal(rel) {
		return filePath
	}
	return filepath.Join(codegen.Gendir, "jsonrpc", rel)
}
