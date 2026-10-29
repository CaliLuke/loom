package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestTransformNamedJSONCollectionBoundary(t *testing.T) {
	for _, test := range []struct {
		name  string
		value expr.DataType
	}{
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			named := &expr.UserTypeExpr{TypeName: "JSONField", AttributeExpr: &expr.AttributeExpr{Type: test.value}}
			source := &expr.AttributeExpr{Type: &expr.Object{{Name: "field", Attribute: &expr.AttributeExpr{Type: named}}}}
			target := &expr.AttributeExpr{Type: &expr.Object{{Name: "field", Attribute: &expr.AttributeExpr{Type: test.value}}}}
			scope := NewNameScope()
			sourceContext := NewAttributeContext(true, false, false, "", scope)
			sourceContext.JSONPresenceTypes = map[string]bool{named.Hash(): true}
			targetContext := NewAttributeContext(false, false, true, "", scope)
			codePlan, err := GoTransform(source, target, "body", "payload", sourceContext, targetContext, "unmarshal", true)
			code := codePlan.Code
			require.NoError(t, err)
			require.Contains(t, code, ".Value()", "named JSON carriers must be unwrapped beneath a native form root")
			require.False(t, sourceContext.JSONPresence)
			require.False(t, sourceContext.CollectionElementPresence)
		})
	}
}

func TestTransformNamedNativeBoundaryInsideJSON(t *testing.T) {
	named := &expr.UserTypeExpr{TypeName: "FormVariant", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}}}}
	source := &expr.AttributeExpr{Type: named}
	target := expr.DupAtt(source)
	scope := NewNameScope()
	sourceContext := NewAttributeContext(true, false, false, "", scope)
	sourceContext.JSONPresence = true
	sourceContext.JSONPresenceTypes = map[string]bool{named.Hash(): false}
	targetContext := NewAttributeContext(false, false, true, "", scope)
	plan, err := GoTransform(source, target, "body", "payload", sourceContext, targetContext, "unmarshal", true)
	require.NoError(t, err)
	require.NotContains(t, plan.Code, ".Value()")
	require.Contains(t, plan.Code, "Name: body.Name")
	require.True(t, sourceContext.JSONPresence)
}
