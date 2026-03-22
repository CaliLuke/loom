package codegen

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/CaliLuke/loom/v3/codegen"
)

var (
	jsonrpcCLIConfigurerDeclPattern = regexp.MustCompile(`([A-Za-z0-9_]+)Configurer \*([A-Za-z0-9_]+)\.ConnConfigurer,`)
	jsonrpcCLIConfigurerUsePattern  = regexp.MustCompile(`, ([A-Za-z0-9_]+)Configurer([,)])`)
)

func rewriteJSONRPCTransportPath(path string) string {
	return strings.Replace(path, "/http/", "/jsonrpc/", 1)
}

func rewriteJSONRPCExampleCLIPath(path string) string {
	return strings.Replace(path, "http.go", "jsonrpc.go", 1)
}

func rewriteJSONRPCExampleServerPath(path string) string {
	return filepath.Join(filepath.Dir(path), "jsonrpc.go")
}

func cloneSection(section codegen.Section) codegen.Section {
	if template, ok := section.(*codegen.SectionTemplate); ok {
		cloned := *template
		return &cloned
	}
	return codegen.NewRawSection(section.SectionName(), renderSectionSource(section))
}

func rewriteJSONRPCSectionSource(section codegen.Section, rewrite func(string) string) codegen.Section {
	if template, ok := section.(*codegen.SectionTemplate); ok {
		cloned := *template
		cloned.Source = rewrite(cloned.Source)
		return &cloned
	}
	return codegen.NewRawSection(section.SectionName(), rewrite(renderSectionSource(section)))
}

func rewriteJSONRPCSectionSources(sections []codegen.Section, rewrite func(string) string) []codegen.Section {
	updated := make([]codegen.Section, 0, len(sections))
	for _, section := range sections {
		if section.SectionName() == "source-header" {
			updated = append(updated, cloneSection(section))
			continue
		}
		updated = append(updated, rewriteJSONRPCSectionSource(section, rewrite))
	}
	return updated
}

func rewriteJSONRPCCLIParseEndpointSource(source string) string {
	source = strings.ReplaceAll(source,
		"{{ .VarName }}Configurer *{{ .PkgName }}.ConnConfigurer,",
		"{{ .VarName }}ConfigFn goahttp.ConnConfigureFunc,")
	source = strings.ReplaceAll(source,
		", {{ .VarName }}Configurer{{ end }}",
		", {{ .VarName }}ConfigFn{{ end }}")
	source = jsonrpcCLIConfigurerDeclPattern.ReplaceAllString(source, `${1}ConfigFn goahttp.ConnConfigureFunc,`)
	source = jsonrpcCLIConfigurerUsePattern.ReplaceAllString(source, `, ${1}ConfigFn${2}`)
	return source
}

func rewriteJSONRPCExampleCLISource(source string) string {
	source = strings.ReplaceAll(source, "doHTTP", "doJSONRPC")
	source = strings.ReplaceAll(source, "httpUsage", "jsonrpcUsage")
	return source
}
