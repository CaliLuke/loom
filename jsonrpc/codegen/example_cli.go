package codegen

import (
	"strings"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	httpcodegen "goa.design/goa/v3/http/codegen"
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
	f.Path = strings.Replace(f.Path, "http.go", "jsonrpc.go", 1)
	updateHeader(f)
	updated := make([]codegen.Section, 0, len(f.AllSections()))
	for _, section := range f.AllSections() {
		if section.SectionName() == "source-header" {
			updated = append(updated, section)
			continue
		}
		source := renderSectionSource(section)
		source = strings.ReplaceAll(source, "doHTTP", "doJSONRPC")
		source = strings.ReplaceAll(source, "httpUsage", "jsonrpcUsage")
		updated = append(updated, codegen.NewRawSection(section.SectionName(), source))
	}
	f.SetSections(updated)
	return f
}
