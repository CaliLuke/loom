package openapiv3

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// Files returns the configured OpenAPI specification files in JSON and YAML formats.
func Files(root *expr.RootExpr) ([]*codegen.File, error) {
	target, err := targetOpenAPIVersion(root.API.Meta)
	if err != nil {
		return nil, err
	}
	spec, warnings, err := newForVersion(root, target)
	if err != nil {
		return nil, err
	}
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
			Warnings: append([]string(nil), warnings...),
		},
		{
			Path:     filepath.Join(codegen.Gendir, "http", "openapi.yaml"),
			Sections: []codegen.Section{codegen.NewRawSection("openapi_v3", yamlSource)},
			Warnings: append([]string(nil), warnings...),
		},
	}, nil
}

func toJSON(meta expr.MetaExpr, d any) (string, error) {
	prefix, p := meta.Last("openapi:json:prefix")
	indent, i := meta.Last("openapi:json:indent")
	options := []json.Options{json.Deterministic(true)}
	if p || i {
		options = append(options, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
	}
	b, err := json.Marshal(d, options...)
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
