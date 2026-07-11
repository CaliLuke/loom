package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ServerTypeFiles returns the JSON-RPC transport type files.
func ServerTypeFiles(genpkg string, services *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.ServerTypeFiles(genpkg, services)
	for _, f := range res {
		updateHeader(f)
		f.Path = rewriteJSONRPCTransportPath(f.Path)
	}
	return res
}
