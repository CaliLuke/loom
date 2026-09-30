package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func applySchemaValidation(s *Schema, attr *expr.AttributeExpr, val *expr.ValidationExpr) {
	if val == nil {
		return
	}
	applySchemaEnums(s, attr, val.Enums())
	applySchemaStringClauses(s, val)
	s.ExclusiveMinimum = val.ExclusiveMinimum
	s.Minimum = val.Minimum
	s.ExclusiveMaximum = val.ExclusiveMaximum
	s.Maximum = val.Maximum
	applySchemaLengths(s, attr, val)
	applySchemaRequired(s, attr, val.Required)
}

func applySchemaEnums(s *Schema, attr *expr.AttributeExpr, enums [][]any) {
	for _, values := range enums {
		if len(values) == 0 {
			s.Not = &Schema{}
			continue
		}
		projected := projectOpenAPIValues(attr, values)
		if s.Enum == nil {
			s.Enum = projected
		} else {
			s.AllOf = append(s.AllOf, &Schema{Enum: projected})
		}
	}
}

func applySchemaStringClauses(s *Schema, val *expr.ValidationExpr) {
	for _, pattern := range val.Patterns() {
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
}

func applySchemaLengths(s *Schema, attr *expr.AttributeExpr, val *expr.ValidationExpr) {
	if val.MinLength != nil {
		if expr.AsArray(attr.Type) != nil {
			s.MinItems = val.MinLength
		} else {
			s.MinLength = val.MinLength
		}
	}
	if val.MaxLength != nil {
		if expr.AsArray(attr.Type) != nil {
			s.MaxItems = val.MaxLength
		} else {
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
