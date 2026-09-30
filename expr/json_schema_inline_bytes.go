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
func inlineByteBounds(attr *AttributeExpr) (byteschema.Bounds, bool, error) {
	visited := make(map[UserType]struct{})
	for current := attr; current != nil; {
		if encodingmeta.Replacement(current.Meta) != "" || encodingmeta.SchemaOverride(current.Meta) {
			return byteschema.Bounds{}, false, nil
		}
		alias, ok := current.Type.(UserType)
		if !ok {
			if current.Type != Bytes {
				return byteschema.Bounds{}, false, nil
			}
			constraints, err := EffectiveConstraintsFor(attr)
			if err != nil {
				return byteschema.Bounds{}, false, err
			}
			validation := constraints.Validation().Lowered()
			return byteschema.Bounds{Minimum: validation.MinLength, Maximum: validation.MaxLength}, true, nil
		}
		if _, exists := visited[alias]; exists {
			return byteschema.Bounds{}, false, nil
		}
		visited[alias] = struct{}{}
		current = alias.Attribute()
	}
	return byteschema.Bounds{}, false, nil
}
