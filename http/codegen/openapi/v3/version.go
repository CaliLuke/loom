package openapiv3

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiversion"
)

type openAPIVersion uint8

type (
	versionRouter struct {
		target      openAPIVersion
		diagnostics []string
	}

	versionRange struct {
		from    openAPIVersion
		through openAPIVersion
	}

	versionedConstructor[T any] struct {
		versions  versionRange
		construct func() (T, []string)
	}

	versionedPass struct {
		versions versionRange
		apply    func() []string
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
	return openAPIVersionForTarget(target)
}

func openAPIVersionForTarget(target openapiversion.Target) (openAPIVersion, error) {
	switch target {
	case openapiversion.Target31:
		return openAPIVersion31, nil
	case openapiversion.Target32:
		return openAPIVersion32, nil
	default:
		return 0, fmt.Errorf("unmapped OpenAPI renderer target %d", target)
	}
}

func renderOpenAPIVersion(router *versionRouter) (string, error) {
	return mustConstruct(
		router,
		"OpenAPI version string",
		versionedConstructor[string]{
			versions: versionRange{from: openAPIVersion31, through: openAPIVersion31},
			construct: func() (string, []string) {
				return OpenAPICompatibilityVersion, nil
			},
		},
		versionedConstructor[string]{
			versions: versionRange{from: openAPIVersion32, through: openAPIVersion32},
			construct: func() (string, []string) {
				return OpenAPIVersion, nil
			},
		},
	)
}

// construct selects the matching constructor with the newest lower bound.
// No match represents an additive feature that is unavailable for the target.
func construct[T any](r *versionRouter, constructors ...versionedConstructor[T]) (T, bool) {
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
		value, warnings := selected.construct()
		r.diagnostics = append(r.diagnostics, warnings...)
		return value, true
	}
	var zero T
	return zero, false
}

func mustConstruct[T any](
	r *versionRouter,
	label string,
	constructors ...versionedConstructor[T],
) (T, error) {
	value, ok := construct(r, constructors...)
	if ok {
		return value, nil
	}
	var zero T
	return zero, fmt.Errorf("no %s for renderer target %d", label, r.target)
}

func (r *versionRouter) runPasses(passes ...versionedPass) {
	for _, pass := range passes {
		if pass.apply == nil || !pass.versions.contains(r.target) {
			continue
		}
		r.diagnostics = append(r.diagnostics, pass.apply()...)
	}
}

func (r *versionRouter) warnings() []string {
	return append([]string(nil), r.diagnostics...)
}

func (r versionRange) contains(target openAPIVersion) bool {
	return (r.from == 0 || target >= r.from) && (r.through == 0 || target <= r.through)
}
