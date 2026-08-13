package openapiv3

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiversion"
)

type openAPIVersion uint8

type (
	versionRouter struct {
		target openAPIVersion
	}

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
	router := versionRouter{target: target}
	value, _ := router.construct(
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

// construct selects the matching constructor with the newest lower bound.
// No match represents an additive feature that is unavailable for the target.
func (r versionRouter) construct[T any](constructors ...versionedConstructor[T]) (T, bool) {
	var selected *versionedConstructor[T]
	for i := range constructors {
		constructor := &constructors[i]
		if constructor.construct == nil || !constructor.versions.contains(r.target) {
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
