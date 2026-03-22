package codegen

import (
	"github.com/CaliLuke/loom/v3/codegen"
	httpcodegen "github.com/CaliLuke/loom/v3/http/codegen"
)

// ClientCLIFiles returns the JSON-RPC transport type files.
func ClientCLIFiles(genpkg string, services *httpcodegen.ServicesData) []*codegen.File {
	res := httpcodegen.ClientCLIFiles(genpkg, services)
	for _, f := range res {
		updateHeader(f)
		f.Path = rewriteJSONRPCTransportPath(f.Path)
		f.SetSections(rewriteJSONRPCCLISections(f.AllSections()))
	}
	return res
}

func rewriteJSONRPCCLISections(sections []codegen.Section) []codegen.Section {
	updated := make([]codegen.Section, 0, len(sections))
	for _, section := range sections {
		if section.SectionName() != "parse-endpoint" {
			updated = append(updated, cloneSection(section))
			continue
		}
		updated = append(updated, rewriteJSONRPCSectionSource(section, rewriteJSONRPCCLIParseEndpointSource))
	}
	return updated
}
