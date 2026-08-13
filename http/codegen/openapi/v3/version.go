package openapiv3

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

type openAPIVersion uint8

type (
	versionRange struct {
		from    openAPIVersion
		through openAPIVersion
	}

	versionedConstructor[T any] struct {
		versions  versionRange
		construct func() T
	}
)

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
	value, ok := meta.Last("openapi:version")
	if !ok || strings.TrimSpace(value) == "" {
		return openAPIVersion32, nil
	}
	switch strings.TrimSpace(value) {
	case "3.1", "3.1.0", "3.1.1", "3.1.2":
		return openAPIVersion31, nil
	case "3.2", OpenAPIVersion:
		return openAPIVersion32, nil
	default:
		return 0, fmt.Errorf("unsupported OpenAPI version %q", value)
	}
}

func renderOpenAPIVersion(target openAPIVersion) string {
	value, _ := constructForVersion(target,
		versionedConstructor[string]{
			versions: versionRange{from: openAPIVersion31, through: openAPIVersion31},
			construct: func() string {
				return OpenAPICompatibilityVersion
			},
		},
		versionedConstructor[string]{
			versions: versionRange{from: openAPIVersion32},
			construct: func() string {
				return OpenAPIVersion
			},
		},
	)
	return value
}

// constructForVersion selects the matching constructor with the newest lower bound.
// No match represents an additive feature that is unavailable for the target.
func constructForVersion[T any](target openAPIVersion, constructors ...versionedConstructor[T]) (T, bool) {
	var selected *versionedConstructor[T]
	for i := range constructors {
		constructor := &constructors[i]
		if constructor.construct == nil || !constructor.versions.contains(target) {
			continue
		}
		if selected == nil || constructor.versions.from > selected.versions.from {
			selected = constructor
		}
	}
	if selected != nil {
		return selected.construct(), true
	}
	var zero T
	return zero, false
}

func (r versionRange) contains(target openAPIVersion) bool {
	return target >= r.from && (r.through == 0 || target <= r.through)
}
