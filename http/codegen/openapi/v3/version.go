package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiversion"
)

type openAPIVersion uint8

const (
	openAPIVersion31 openAPIVersion = iota + 1
	openAPIVersion32
)

const (
	// OpenAPIVersion is the default OpenAPI specification version emitted by Loom.
	OpenAPIVersion = "3.2.0"
	// OpenAPICompatibilityVersion is the optional compatibility target that omits 3.2-only sections.
	OpenAPICompatibilityVersion = "3.1.1"
)

func targetOpenAPIVersion(meta expr.MetaExpr) (openAPIVersion, error) {
	value, _ := meta.Last("openapi:version")
	target, err := openapiversion.Parse(value)
	if err != nil {
		return 0, err
	}
	switch target {
	case openapiversion.Target31:
		return openAPIVersion31, nil
	case openapiversion.Target32:
		return openAPIVersion32, nil
	}
	return openAPIVersion32, nil
}

func renderOpenAPIVersion(target openAPIVersion) string {
	if target == openAPIVersion31 {
		return OpenAPICompatibilityVersion
	}
	return OpenAPIVersion
}
