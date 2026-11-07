package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/encodingmeta"
)

func TestIntrinsicUnionNullFieldStorage(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first *expr.AttributeExpr
	}{
		{"any branch", &expr.AttributeExpr{Type: expr.Any}},
		{"nullable branch", &expr.AttributeExpr{Type: expr.String, Nullable: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			union := &expr.Union{
				TypeName: "Choice",
				Untagged: true,
				Values: []*expr.NamedAttributeExpr{
					{Name: "First", Attribute: tc.first},
					{Name: "Second", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
				},
			}
			attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
				TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: union},
			}}
			parent := &expr.AttributeExpr{
				Type: &expr.Object{
					{Name: "required", Attribute: attribute},
					{Name: "optional", Attribute: attribute},
				},
				Validation: &expr.ValidationExpr{Required: []string{"required"}},
			}
			require.True(t, expr.AllowsNull(attribute))
			code := NewNameScope().GoTypeDef(parent, false, false)
			require.Contains(t, code, "Required Choice `json:\"required\"`")
			require.Contains(t, code, "Optional Choice `json:\"optional,omitzero\"`")
			for _, name := range []string{"required", "optional"} {
				layout := expr.ServiceFieldLayout(parent, name, attribute, false, false)
				require.False(t, layout.Pointer)
				require.Equal(t, encodingmeta.JSONPresence, layout.JSON.Storage)
			}
			for _, json := range []bool{false, true} {
				context := &AttributeContext{JSONPresence: json}
				for _, name := range []string{"required", "optional"} {
					require.Equal(t, NativePresence, context.FieldPresence(expr.NewObjectFields(parent), name, attribute))
				}
				outer := expr.DupAtt(attribute)
				outer.Nullable = true
				require.Equal(t, NullablePresence, context.FieldPresence(expr.NewObjectFields(parent), "optional", outer))
				require.False(t, expr.HasNativeJSONPresence(outer))
			}
			// A union that cannot select null keeps ordinary optional storage.
			union.Values[0].Attribute = &expr.AttributeExpr{Type: expr.String}
			require.False(t, expr.HasNativeJSONPresence(attribute))
			require.True(t, expr.ServiceFieldLayout(parent, "optional", attribute, false, false).Pointer)
			require.Equal(t, OptionalPresence, (&AttributeContext{JSONPresence: true}).FieldPresence(expr.NewObjectFields(parent), "optional", attribute))
		})
	}
}
