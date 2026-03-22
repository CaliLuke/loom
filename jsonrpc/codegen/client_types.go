package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ClientTypeFiles returns the JSON-RPC transport type files.
func ClientTypeFiles(genpkg string, services *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.ClientTypeFiles(genpkg, services)
	for _, f := range res {
		f.Path = strings.Replace(f.Path, "/http/", "/jsonrpc/", 1)
	}
	return res
}
