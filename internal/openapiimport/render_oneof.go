package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapiref"
)

// oneOfExpression preserves branch attributes instead of reducing an inline
// schema to its constructor. Reference-only unions retain their compact form.
func (r *renderer) oneOfExpression(schema *Schema, path string) (string, bool, error) {
	expressions := make([]string, len(schema.OneOf))
	bases := make([]string, len(schema.OneOf))
	keys := make([]string, len(schema.OneOf))
	compact := true
	for index, branch := range schema.OneOf {
		expression, object, err := r.schemaExpression(branch, fmt.Sprintf("%s/oneOf/%d", path, index))
		if err != nil {
			return "", false, err
		}
		if object {
			return "", false, fmt.Errorf("render OpenAPI design: %s/oneOf/%d object branch lacks a concrete name", path, index)
		}
		expressions[index] = expression
		bare := *branch
		if bare.Format == "" && (bare.Type == "integer" || bare.Type == "number") {
			bare.inheritedFormat = true
		}
		primitive := branch.Type == "string" || branch.Type == "integer" || branch.Type == "number" || branch.Type == "boolean"
		compact = compact && (branch.Ref != "" && !r.hasSchemaBlock(branch) || primitive && !r.hasSchemaBlock(&bare))
		bases[index], err = oneOfBranchName(branch, expression)
		if err != nil {
			return "", false, err
		}
		keys[index] = fmt.Sprintf("%08d", index)
	}
	// The compact public constructor requires two types. Retained singleton
	// unions use the function form so occurrence closure is not discarded.
	if len(schema.OneOf) >= 2 && (compact || len(schema.MediaTypes) > 0) {
		return "OneOf(" + strings.Join(expressions, ", ") + ")", false, nil
	}
	names := expr.UniqueStableNames(bases, keys, func(base string, ordinal int) string {
		return fmt.Sprintf("%s%d", base, ordinal)
	})
	child := r.childRenderer()
	child.open("OneOf(func()")
	if len(schema.MediaTypes) == 0 {
		child.line("Untagged()")
	}
	for index, branch := range schema.OneOf {
		if err := child.attribute(names[index], branch, nil, fmt.Sprintf("%s/oneOf/%d", path, index)); err != nil {
			return "", false, err
		}
	}
	child.close()
	return strings.TrimSpace(child.builder.String()), false, nil
}

// oneOfBranchName supplies stable function-constructor branch names without
// changing the expression that renders the branch's complete schema.
func oneOfBranchName(branch *Schema, expression string) (string, error) {
	switch {
	case branch.Ref != "":
		name, err := openapiref.Name(branch.Ref, "#/components/schemas/")
		if err != nil {
			return "", err
		}
		return expr.Title(name), nil
	case len(branch.OneOf) > 0:
		return "OneOf", nil
	case branch.Type == "array":
		return "Array", nil
	case branch.Type == "object":
		return "Map", nil
	default:
		return expression, nil
	}
}

func (r *renderer) unionSchema(schema *Schema) bool {
	seen := make(map[*Schema]bool)
	for schema != nil && !seen[schema] {
		seen[schema] = true
		if len(schema.OneOf) > 0 {
			return true
		}
		if schema.Ref == "" {
			return false
		}
		resolved, err := r.resolveSchemaReference(schema, "")
		if err != nil {
			return false
		}
		schema = resolved
	}
	return false
}
