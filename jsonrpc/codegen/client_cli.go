package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ClientCLIFiles returns the JSON-RPC transport type files.
func ClientCLIFiles(genpkg string, services *httpcodegen.ServicesData) []*codegen.File {
	return httpcodegen.ClientCLIFilesForTransport(genpkg, services, httpcodegen.ClientCLITransport{
		PathName:    "jsonrpc",
		DisplayName: "JSON-RPC",
		StreamingConfigurerName: func(serviceVar string) string {
			return serviceVar + "ConfigFn"
		},
		StreamingConfigurerType: func(string) string {
			return "loomhttp.ConnConfigureFunc"
		},
	})
}
