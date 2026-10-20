package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionNullAdmission(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		nullable  bool
	}{
		{"nullable string", &expr.AttributeExpr{Type: expr.String, Nullable: true}, true},
		{"inherited nullable", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "NullableString", UID: "nullable-string", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Nullable: true}}}, true},
		{"any", &expr.AttributeExpr{Type: expr.Any}, true},
		{"ordinary string", &expr.AttributeExpr{Type: expr.String}, false},
		{"ordinary object", &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Object", UID: "object", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			union := &expr.Union{TypeName: "Selection", Values: []*expr.NamedAttributeExpr{{Name: "branch", Attribute: tc.attribute}}}
			for _, view := range []bool{false, true} {
				t.Run(fmt.Sprintf("view=%t", view), func(t *testing.T) {
					scope := codegen.NewNameScope()
					var data *UnionTypeData
					if view {
						data = buildViewUnionTypeData(union, scope, nil)
					} else {
						data = buildUnionTypeData(union, scope, nil)
					}
					body := uniongen.UnmarshalJSONBody(&data.Type)
					require.Contains(t, body, "len(raw.Value) == 0")
					require.Equal(t, !tc.nullable, strings.Contains(body, `|| string(raw.Value) == "null"`), "view=%t: %s", view, body)
				})
			}
		})
	}
}
