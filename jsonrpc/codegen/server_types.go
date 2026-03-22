package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/v3/codegen"
	httpcodegen "github.com/CaliLuke/loom/v3/http/codegen"
)

// ServerTypeFiles returns the JSON-RPC transport type files.
func ServerTypeFiles(genpkg string, services *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.ServerTypeFiles(genpkg, services)
	for _, f := range res {
		updateHeader(f)
		f.Path = strings.Replace(f.Path, "/http/", "/jsonrpc/", 1)
	}
	return res
}
