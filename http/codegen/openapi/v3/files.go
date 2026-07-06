package openapiv3

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// Files returns the OpenAPI 3.1 specification files in JSON and YAML formats.
func Files(root *expr.RootExpr) ([]*codegen.File, error) {
	spec := New(root)
	jsonSource, err := toJSON(root.API.Meta, spec)
	if err != nil {
		return nil, err
	}
	yamlSource, err := toYAML(spec)
	if err != nil {
		return nil, err
	}

	return []*codegen.File{
		{
			Path:     filepath.Join(codegen.Gendir, "http", "openapi.json"),
			Sections: []codegen.Section{codegen.NewRawSection("openapi_v3", jsonSource)},
		},
		{
			Path:     filepath.Join(codegen.Gendir, "http", "openapi.yaml"),
			Sections: []codegen.Section{codegen.NewRawSection("openapi_v3", yamlSource)},
		},
	}, nil
}

func toJSON(meta expr.MetaExpr, d any) (string, error) {
	prefix, p := meta.Last("openapi:json:prefix")
	indent, i := meta.Last("openapi:json:indent")
	marshal := json.Marshal
	if p || i {
		marshal = func(v any) ([]byte, error) {
			return json.MarshalIndent(v, prefix, indent)
		}
	}
	b, err := marshal(d)
	if err != nil {
		return "", fmt.Errorf("openapi json: %w", err)
	}
	return string(b), nil
}

func toYAML(d any) (string, error) {
	b, err := yaml.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("openapi yaml: %w", err)
	}
	return string(b), nil
}
