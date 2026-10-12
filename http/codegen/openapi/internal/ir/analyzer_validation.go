package ir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func applySchemaValidation(s *Schema, attr *expr.AttributeExpr, val *expr.ValidationExpr, enums [][]any) error {
	if val == nil {
		return nil
	}
	applySchemaEnums(s, enums)
	if err := applySchemaStringClauses(s, val); err != nil {
		return err
	}
	s.ExclusiveMinimum = val.ExclusiveMinimum
	s.Minimum = val.Minimum
	s.ExclusiveMaximum = val.ExclusiveMaximum
	s.Maximum = val.Maximum
	applySchemaLengths(s, attr, val)
	applySchemaRequired(s, attr, val.Required)
	return nil
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
		pattern, err := projectPattern(authored)
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
