package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// ExampleCLIFiles returns example JSON-RPC client CLI implementation.
func ExampleCLIFiles(genpkg string, data *httpcodegen.ServicesData) []*codegen.File {
	var fw []*codegen.File
	for _, svr := range data.Root.API.Servers {
		if m := exampleCLI(genpkg, data, svr); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleCLI(genpkg string, data *httpcodegen.ServicesData, svr *expr.ServerExpr) *codegen.File {
	f := httpcodegen.ExampleCLI(genpkg, svr, data)
	if f == nil {
		return nil
	}
	f.Path = rewriteJSONRPCExampleCLIPath(f.Path)
	updateHeader(f)
	f.SetSections(rewriteJSONRPCSectionSources(f.AllSections(), rewriteJSONRPCExampleCLISource))
	return f
}
