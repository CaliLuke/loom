package openapiv3

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapimeta"
)

const (
	openAPIOutputBoth = "both"
	openAPIOutputJSON = "json"
	openAPIOutputYAML = "yaml"
)

// Files returns the configured OpenAPI specification files.
func Files(root *expr.RootExpr) ([]*codegen.File, error) {
	if err := validateTagMetadata(root); err != nil {
		return nil, err
	}
	target, err := targetOpenAPIVersion(root.API.Meta)
	if err != nil {
		return nil, err
	}
	spec, warnings, err := newForVersion(root, target)
	if err != nil {
		return nil, err
	}
	output, err := selectedOpenAPIOutput(root.API.Meta)
	if err != nil {
		return nil, err
	}

	var files []*codegen.File
	if output == openAPIOutputJSON || output == openAPIOutputBoth {
		jsonSource, err := toJSON(root.API.Meta, spec)
		if err != nil {
			return nil, err
		}
		file := openAPIFile("json", jsonSource, warnings)
		if output == openAPIOutputJSON {
			file.RemovePaths = []string{openAPIPath("yaml")}
		}
		files = append(files, file)
	}
	if output == openAPIOutputYAML || output == openAPIOutputBoth {
		yamlSource, err := toYAML(spec)
		if err != nil {
			return nil, err
		}
		file := openAPIFile("yaml", yamlSource, warnings)
		if output == openAPIOutputYAML {
			file.RemovePaths = []string{openAPIPath("json")}
		}
		files = append(files, file)
	}
	return files, nil
}

func validateTagMetadata(root *expr.RootExpr) error {
	if root == nil || root.API == nil {
		return nil
	}
	if err := validateTagMetadataAt("API", root.API.Meta); err != nil {
		return err
	}
	if root.API.HTTP == nil {
		return nil
	}
	for _, service := range root.API.HTTP.Services {
		if service == nil {
			continue
		}
		if err := validateTagMetadataAt("HTTP service", service.Meta); err != nil {
			return err
		}
		if service.ServiceExpr != nil {
			if err := validateTagMetadataAt("service", service.ServiceExpr.Meta); err != nil {
				return err
			}
			for _, method := range service.ServiceExpr.Methods {
				if method != nil {
					if err := validateTagMetadataAt("method "+method.Name, method.Meta); err != nil {
						return err
					}
				}
			}
		}
		for _, endpoint := range service.HTTPEndpoints {
			if endpoint == nil {
				continue
			}
			if err := validateTagMetadataAt("HTTP endpoint "+endpoint.Name(), endpoint.Meta); err != nil {
				return err
			}
			for _, route := range endpoint.Routes {
				if route != nil {
					if err := validateTagMetadataAt("HTTP route "+route.Method+" "+route.Path, route.Meta); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func validateTagMetadataAt(owner string, metadata expr.MetaExpr) error {
	if err := openapimeta.ValidateTagMetadata(metadata); err != nil {
		return fmt.Errorf("invalid OpenAPI tag metadata on %s: %w", owner, err)
	}
	return nil
}

func toJSON(meta expr.MetaExpr, d any) (string, error) {
	prefix, _ := meta.Last("openapi:json:prefix")
	indent, i := meta.Last("openapi:json:indent")
	if !i || indent == "" {
		indent = "  "
	}
	options := []json.Options{
		json.Deterministic(true),
		jsontext.WithIndentPrefix(prefix),
		jsontext.WithIndent(indent),
	}
	b, err := json.Marshal(d, options...)
	if err != nil {
		return "", fmt.Errorf("openapi json: %w", err)
	}
	return string(b), nil
}

func selectedOpenAPIOutput(meta expr.MetaExpr) (string, error) {
	output, ok := meta.Last("openapi:output")
	if !ok {
		return openAPIOutputBoth, nil
	}
	switch output {
	case openAPIOutputJSON, openAPIOutputYAML, openAPIOutputBoth:
		return output, nil
	default:
		return "", fmt.Errorf("openapi output format %q must be one of json, yaml, or both", output)
	}
}

func openAPIFile(extension, source string, warnings []string) *codegen.File {
	return &codegen.File{
		Path:     openAPIPath(extension),
		Sections: []codegen.Section{codegen.NewRawSection("openapi_v3", source)},
		Warnings: append([]string(nil), warnings...),
	}
}

func openAPIPath(extension string) string {
	return filepath.Join(codegen.Gendir, "http", "openapi."+extension)
}

func toYAML(d any) (string, error) {
	b, err := yaml.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("openapi yaml: %w", err)
	}
	return string(b), nil
}
