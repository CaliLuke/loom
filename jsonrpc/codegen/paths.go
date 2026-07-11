package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// PathFiles returns the service path files.
func PathFiles(data *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.PathFiles(data)
	for _, f := range res {
		updateHeader(f)
		f.Path = rewriteJSONRPCTransportPath(f.Path)
	}
	return res
}
