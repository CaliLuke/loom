package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

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
