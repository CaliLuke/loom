package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestProjectedTransformPreservesRequiredPresence checks conversion into the
// pointer carrier used before selected-view validation. Full-type requiredness
// cannot establish that a field was supplied in a particular view.
func TestProjectedTransformPreservesRequiredPresence(t *testing.T) {
	cases := []struct {
		name string
		typ  expr.DataType
	}{
		{"object", &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
		{"named_scalar", &expr.UserTypeExpr{TypeName: "Label", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{
				Type:       &expr.Object{{Name: "data", Attribute: &expr.AttributeExpr{Type: c.typ}}},
				Validation: &expr.ValidationExpr{Required: []string{"data"}},
			}
			for _, useDefault := range []bool{false, true} {
				scope := NewNameScope()
				source := NewAttributeContext(true, false, useDefault, "", scope)
				target := NewAttributeContext(true, false, true, "", scope)
				code, _, err := GoTransform(attribute, attribute, "source", "target", source, target, "convert", true)
				require.NoError(t, err)
				require.Contains(t, code, "if source.Data != nil {", "source defaults=%t", useDefault)
				require.NotContains(t, code, "else", "an absent field must remain absent before validation")
			}
		})
	}
}
