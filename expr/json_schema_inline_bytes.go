package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/internal/byteschema"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

func buildInlineByteSchema(attr *AttributeExpr, visited map[any]struct{}, context inlineSchemaContext, bounds byteschema.Bounds) (*InlineSchema, error) {
	context.byteLengthsHandled = true
	schema, err := buildInlineJSONSchema(attr, visited, context)
	if err != nil {
		return nil, err
	}
	if err := applyInlineByteConstraint(schema, bounds, AllowsNull(attr)); err != nil {
		return nil, err
	}
	return schema, nil
}

func applyInlineByteConstraint(schema *InlineSchema, bounds byteschema.Bounds, nullable bool) error {
	constraint, err := byteschema.Project(bounds.Minimum, bounds.Maximum)
	if err != nil {
		return fmt.Errorf("Bytes length: %w", err)
	}
	projected := &InlineSchema{}
	if constraint.Unsatisfiable {
		projected.Not = &InlineSchema{}
	} else {
		projected.Not = &InlineSchema{Pattern: constraint.ForbiddenPattern}
		projected.AnyOf = make([]*InlineSchema, 0, len(constraint.Branches))
		for _, branch := range constraint.Branches {
			minimum := branch.MinLength
			projected.AnyOf = append(projected.AnyOf, &InlineSchema{
				Pattern: branch.Pattern, MinLength: &minimum, MaxLength: branch.MaxLength,
			})
		}
	}
	if nullable {
		projected = nullableInlineJSONSchema(projected)
	}
	// Byte bounds describe decoded data, so they cannot remain as direct
	// string lengths. Conjunction preserves every existing enum and alias gate.
	schema.MinLength, schema.MaxLength = nil, nil
	schema.ContentEncoding = "base64"
	schema.AllOf = append(schema.AllOf, projected)
	return nil
}

// inlineByteBounds collects one effective Bytes occurrence before emitting any
// length keywords. Recursive aliases remain the existing builder's error path.
func inlineByteBounds(attr *AttributeExpr) (byteschema.Bounds, bool) {
	var bounds []byteschema.Bounds
	visited := make(map[UserType]struct{})
	for attr != nil {
		if encodingmeta.Replacement(attr.Meta) != "" || encodingmeta.SchemaOverride(attr.Meta) {
			return byteschema.Bounds{}, false
		}
		if attr.Validation != nil {
			bounds = append(bounds, byteschema.Bounds{Minimum: attr.Validation.MinLength, Maximum: attr.Validation.MaxLength})
		}
		alias, ok := attr.Type.(UserType)
		if !ok {
			return byteschema.Intersect(bounds), attr.Type == Bytes
		}
		if _, exists := visited[alias]; exists {
			return byteschema.Bounds{}, false
		}
		visited[alias] = struct{}{}
		attr = alias.Attribute()
	}
	return byteschema.Bounds{}, false
}
