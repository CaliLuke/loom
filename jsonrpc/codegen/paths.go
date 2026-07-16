package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// PathFiles returns the service path files.
func PathFiles(data *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.PathFiles(data)
	for _, f := range res {
		updateHeader(f)
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
