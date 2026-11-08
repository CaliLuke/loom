package ir

import (
	"fmt"
	"slices"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/internal/patternsyntax"
)

func applySchemaValidation(s *Schema, attr *expr.AttributeExpr, val *expr.ValidationExpr, enums [][]any) error {
	applySchemaEnums(s, enums)
	applySchemaRequired(s, attr, schemaRequiredNames(attr))
	if val == nil {
		return nil
	}
	if err := applySchemaStringClauses(s, val); err != nil {
		return err
	}
	s.ExclusiveMinimum = val.ExclusiveMinimum
	s.Minimum = val.Minimum
	s.ExclusiveMaximum = val.ExclusiveMaximum
	s.Maximum = val.Maximum
	applySchemaLengths(s, attr, val)
	return nil
}

// schemaRequiredNames keeps reference-local requirements on named occurrences,
// while a physical object publishes the effective requirements of its retained
// members. Composition snapshots can carry only local Validation; transport
// projections can also remove members that their constraint parents still own.
// Read the shared requiredness owner without recapturing descendant value graphs;
// typed additional keys remain valid requirements.
func schemaRequiredNames(attr *expr.AttributeExpr) []string {
	if _, object := attr.Type.(*expr.Object); !object {
		if attr.Validation == nil {
			return nil
		}
		return attr.Validation.Required
	}
	required := attr.AllRequired()
	typed := attr.TypedObjectDeclaration() != nil
	names := make([]string, 0, len(required))
	for _, name := range required {
		if slices.Contains(attr.Meta[usageOmittedAttributesMeta], name) {
			continue
		}
		if attr.Find(name) != nil || typed {
			names = append(names, name)
		}
	}
	return names
}

func applySchemaEnums(s *Schema, enums [][]any) {
	for _, values := range enums {
		if len(values) == 0 {
			s.Not = &Schema{}
			continue
		}
		if s.Enum == nil {
			s.Enum = values
		} else {
			s.AllOf = append(s.AllOf, &Schema{Enum: values})
		}
	}
}

func applySchemaStringClauses(s *Schema, val *expr.ValidationExpr) error {
	for _, authored := range val.Patterns() {
		pattern, err := patternsyntax.ToECMA(authored)
		if err != nil {
			return fmt.Errorf("OpenAPI pattern %q cannot preserve Go validation: %w", authored, err)
		}
		if s.Pattern == "" {
			s.Pattern = pattern
		} else if s.Pattern != pattern {
			s.AllOf = append(s.AllOf, &Schema{Pattern: pattern})
		}
	}
	for _, format := range val.Formats() {
		if s.Format == "" {
			s.Format = string(format)
		} else if s.Format != string(format) {
			s.AllOf = append(s.AllOf, &Schema{Format: string(format)})
		}
	}
	return nil
}

func applySchemaLengths(s *Schema, attr *expr.AttributeExpr, val *expr.ValidationExpr) {
	if val.MinLength != nil {
		switch {
		case expr.AsArray(attr.Type) != nil:
			s.MinItems = val.MinLength
		case expr.AsMap(attr.Type) != nil:
			s.MinProperties = val.MinLength
		default:
			s.MinLength = val.MinLength
		}
	}
	if val.MaxLength != nil {
		switch {
		case expr.AsArray(attr.Type) != nil:
			s.MaxItems = val.MaxLength
		case expr.AsMap(attr.Type) != nil:
			s.MaxProperties = val.MaxLength
		default:
			s.MaxLength = val.MaxLength
		}
	}
}

func applySchemaRequired(s *Schema, attr *expr.AttributeExpr, requiredNames []string) {
	for _, required := range requiredNames {
		child := attr.Find(required)
		if child != nil {
			if !openapi.MustGenerate(child.Meta) {
				continue
			}
			required = expr.JSONFieldName(expr.ElementName(required), child)
			if required == "-" {
				continue
			}
		}
		s.Required = append(s.Required, required)
	}
}
