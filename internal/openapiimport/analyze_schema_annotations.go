package openapiimport

import (
	"maps"

	"github.com/CaliLuke/libopenapi/datamodel/high/base"
)

// schemaAnnotations composes one authored layer after structural normalization.
// Positive annotations on an admitted branch survive enclosing layers; nonempty
// outer labels and extension keys take precedence without changing branch maps.
func (a *analyzer) schemaAnnotations(schema *Schema, source *base.Schema, path string) {
	if source.Title != "" {
		schema.Title = source.Title
	}
	if source.Description != "" {
		schema.Description = source.Description
	}
	schema.Deprecated = schema.Deprecated || source.Deprecated != nil && *source.Deprecated
	schema.ReadOnly = schema.ReadOnly || source.ReadOnly != nil && *source.ReadOnly
	schema.WriteOnly = schema.WriteOnly || source.WriteOnly != nil && *source.WriteOnly
	outer := a.extensions(path, source.Extensions)
	schema.Extensions = maps.Clone(schema.Extensions)
	if len(outer) > 0 {
		if schema.Extensions == nil {
			schema.Extensions = outer
		} else {
			maps.Copy(schema.Extensions, outer)
		}
	}
}
