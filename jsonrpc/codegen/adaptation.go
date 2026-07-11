package codegen

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

var (
	jsonrpcCLIConfigurerDeclPattern = regexp.MustCompile(`([A-Za-z0-9_]+)Configurer \*([A-Za-z0-9_]+)\.ConnConfigurer([,)])`)
	jsonrpcCLIConfigurerUsePattern  = regexp.MustCompile(`, ([A-Za-z0-9_]+)Configurer([,)])`)
)

func rewriteJSONRPCTransportPath(filePath string) string {
	separator := "/"
	if strings.Contains(filePath, `\`) && !strings.Contains(filePath, "/") {
		separator = `\`
	}
	normalized := strings.ReplaceAll(filePath, `\`, "/")
	rewritten := strings.Replace(normalized, "/http/", "/jsonrpc/", 1)
	if separator == `\` {
		return strings.ReplaceAll(rewritten, "/", `\`)
	}
	return rewritten
}

func rewriteJSONRPCExampleCLIPath(path string) string {
	return strings.Replace(path, "http.go", "jsonrpc.go", 1)
}

func rewriteJSONRPCExampleServerPath(path string) string {
	return filepath.Join(filepath.Dir(path), "jsonrpc.go")
}

func cloneSection(section codegen.Section) codegen.Section {
	switch s := section.(type) {
	case *codegen.SectionTemplate:
		cloned := *s
		return &cloned
	case *codegen.TextTemplateSection:
		cloned := *s
		return &cloned
	case *codegen.JenniferSection:
		cloned := *s
		return &cloned
	case *codegen.RenderSection:
		cloned := *s
		return &cloned
	case *codegen.RawSection:
		cloned := *s
		return &cloned
	default:
		return codegen.NewRenderSection(section.SectionName(), func() string {
			return renderSectionSource(section)
		})
	}
}

func rewriteJSONRPCSectionSource(section codegen.Section, rewrite func(string) string) codegen.Section {
	switch s := section.(type) {
	case *codegen.SectionTemplate:
		cloned := *s
		cloned.Source = rewrite(cloned.Source)
		return &cloned
	case *codegen.TextTemplateSection:
		cloned := *s
		cloned.Source = rewrite(cloned.Source)
		return &cloned
	case *codegen.RawSection:
		cloned := *s
		cloned.Source = rewrite(cloned.Source)
		return &cloned
	case *codegen.RenderSection:
		cloned := *s
		original := cloned.Render
		cloned.Render = func() string {
			if original == nil {
				return ""
			}
			return rewrite(original())
		}
		return &cloned
	case *codegen.JenniferSection:
		cloned := *s
		return codegen.NewRenderSection(cloned.SectionName(), func() string {
			return rewrite(renderSectionSource(&cloned))
		})
	default:
		return codegen.NewRenderSection(section.SectionName(), func() string {
			return rewrite(renderSectionSource(section))
		})
	}
}

func renameJSONRPCSection(section codegen.Section, name string) codegen.Section {
	switch s := section.(type) {
	case *codegen.SectionTemplate:
		cloned := *s
		cloned.Name = name
		return &cloned
	case *codegen.TextTemplateSection:
		cloned := *s
		cloned.Name = name
		return &cloned
	case *codegen.RawSection:
		cloned := *s
		cloned.Name = name
		return &cloned
	case *codegen.RenderSection:
		cloned := *s
		cloned.Name = name
		return &cloned
	case *codegen.JenniferSection:
		cloned := *s
		cloned.Name = name
		return &cloned
	default:
		return codegen.NewRenderSection(name, func() string {
			return renderSectionSource(section)
		})
	}
}

func endpointDataForSection(section codegen.Section) (*httpcodegen.EndpointData, bool) {
	switch s := section.(type) {
	case *codegen.SectionTemplate:
		data, ok := s.Data.(*httpcodegen.EndpointData)
		return data, ok
	case *codegen.TextTemplateSection:
		data, ok := s.Data.(*httpcodegen.EndpointData)
		return data, ok
	default:
		return nil, false
	}
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
		"{{ .VarName }}ConfigFn loomhttp.ConnConfigureFunc,")
	source = strings.ReplaceAll(source,
		", {{ .VarName }}Configurer{{ end }}",
		", {{ .VarName }}ConfigFn{{ end }}")
	source = jsonrpcCLIConfigurerDeclPattern.ReplaceAllString(source, `${1}ConfigFn loomhttp.ConnConfigureFunc${3}`)
	source = jsonrpcCLIConfigurerUsePattern.ReplaceAllString(source, `, ${1}ConfigFn${2}`)
	return source
}

func rewriteJSONRPCExampleCLISource(source string) string {
	source = strings.ReplaceAll(source, "doHTTP", "doJSONRPC")
	source = strings.ReplaceAll(source, "httpUsage", "jsonrpcUsage")
	return source
}
