package unionjson

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestAdditionalShapeRetainsNestedPolicy(t *testing.T) {
	inner := &expr.AttributeExpr{Type: &expr.Object{{Name: "hidden", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"openapi:generate": {"false"}}}}}, AdditionalAttributes: &expr.AttributeExpr{Type: expr.Bytes}}
	source := &expr.AttributeExpr{Type: &expr.Object{}, AdditionalAttributes: inner}
	plan, err := Plan(source, expr.ElementName, true, false)
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	matched, err := shape.Match([]byte(`{"extra":{"hidden":"aGk="}}`))
	require.NoError(t, err)
	require.False(t, matched)
	matched, err = shape.Match([]byte(`{"extra":{"other":"aGk="}}`))
	require.NoError(t, err)
	require.True(t, matched)
	var rendered strings.Builder
	renderShape(&rendered, "shape", shape)
	require.Contains(t, rendered.String(), "Additional: &shapeNodes[")
	require.Contains(t, rendered.String(), `ForbiddenMembers: []string{"hidden"}`)
}

func TestAdditionalShapeRetainsAliasRequirements(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}, AdditionalAttributes: &expr.AttributeExpr{Type: expr.Bytes}}}
	source := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{Required: []string{"x"}}}
	plan, err := Plan(source, expr.ElementName, true, false)
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	matched, err := shape.Match([]byte(`{}`))
	require.NoError(t, err)
	require.False(t, matched)
	var rendered strings.Builder
	renderShape(&rendered, "shape", shape)
	require.Contains(t, rendered.String(), `RequiredMembers: []string{"x"}`)
}
