package unionjson

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestPlanTypedAdditionalMembers(t *testing.T) {
	nested := &expr.AttributeExpr{Type: &expr.Object{{Name: "value:wire", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Validation: &expr.ValidationExpr{Required: []string{"value:wire"}}}
	source := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "hidden", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"openapi:generate": {"false"}}}},
	}, AdditionalAttributes: nested, Validation: &expr.ValidationExpr{Required: []string{"required"}}}
	plan, err := Plan(source, transportJSONFieldName, true, true)
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"required":{"wire":"ok"},"extra":{"wire":"ok"}}`, true},
		{`{"required":{"wire":"ok"},"extra":{"value":"bad"}}`, false},
		{`{"required":{"wire":"ok"},"extra":{"wire":"ok","other":1}}`, false},
		{`{"required":{"wire":"ok"},"hidden":{"wire":"bad"}}`, false},
		{`{"extra":{"wire":"ok"}}`, false},
	} {
		got, err := shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, got, tc.wire)
	}
	require.Equal(t, expr.ObjectOpennessUnset, nested.ObjectOpenness(), "projection must not mutate source")
}

func TestRenderShapeAdditionalMembers(t *testing.T) {
	child := &loom.JSONShape{Kind: "integer"}
	root := &loom.JSONShape{Kind: "object", Additional: child, ForbiddenMembers: []string{"removed"}, Fields: []loom.JSONShapeField{{Name: "required", Required: true, Shape: child}}}
	var output strings.Builder
	RenderShape(&output, "shape", root)
	require.Contains(t, output.String(), "make([]loom.JSONShape, 2)")
	require.Contains(t, output.String(), "Additional: &shapeNodes[1]")
	require.Contains(t, output.String(), `ForbiddenMembers: []string{"removed"}`)
	require.Contains(t, output.String(), `Name: "required", Required: true, Shape: &shapeNodes[1]`)
}

func transportJSONFieldName(name string, attribute *expr.AttributeExpr) string {
	return expr.JSONFieldName(expr.ElementName(name), attribute)
}
