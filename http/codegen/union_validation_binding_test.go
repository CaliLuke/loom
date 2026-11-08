package codegen

import (
	"fmt"
	"testing"

	cg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestHTTPUnionValidationBelongsToPhysicalDeclaration(t *testing.T) {
	for _, intrinsic := range []bool{false, true} {
		for _, first := range []bool{false, true} {
			t.Run(fmt.Sprintf("intrinsic=%t/pointer-first=%t", intrinsic, first), func(t *testing.T) {
				scope := cg.NewNameScope()
				child := &expr.UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &expr.AttributeExpr{
					Type:       &expr.Object{{Name: "kind", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"cat"}}}}},
					Nullable:   intrinsic,
					Validation: &expr.ValidationExpr{Required: []string{"kind"}},
				}}
				union := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "Record", Attribute: &expr.AttributeExpr{Type: child, Nullable: !intrinsic}}}}
				attribute := &expr.AttributeExpr{Type: union}
				allocated := make(map[string]*uniongen.Type)
				published := make(map[bool]*uniongen.Type)
				for _, pointer := range []bool{first, !first} {
					context := bodyCodecContext(scope, !pointer)
					context.JSONPresenceTypes = map[string]bool{child.Hash(): false}
					context.PresencePointerTypes = map[string]bool{child.Hash(): pointer}
					context.PresenceUseDefaultTypes = map[string]bool{child.Hash(): !pointer}
					members := make(map[string]*uniongen.Type)
					collectHTTPUnionTypes(attribute, scope, context, members, make(map[string]struct{}), false, allocated)
					published[pointer] = members[scope.GoTypeName(attribute)]
				}
				require.Len(t, allocated, 1)
				require.Equal(t, published[true].Name, published[false].Name)
				require.Equal(t, published[true].Fields[0].FieldType, published[false].Fields[0].FieldType)
				for _, pointer := range []bool{false, true} {
					declaration := published[pointer]
					operand := "actual.Kind"
					if pointer {
						operand = "*actual.Kind"
					}
					require.Contains(t, declaration.Fields[0].ValidateCode, operand+` == "cat"`)
					require.Equal(t, declaration.Fields[0].ValidateCode, declaration.JSON.Branches[0].Validate)
				}
				require.NotSame(t, published[true].JSON, published[false].JSON)
				require.NotSame(t, published[true].JSON.Branches[0], published[false].JSON.Branches[0])
			})
		}
	}
}

func TestHTTPUnionValidationPreservesBranchStorage(t *testing.T) {
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		operand   string
	}{
		{"scalar", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"cat"}}}, "v"},
		{"nullable scalar", &expr.AttributeExpr{Type: expr.String, Nullable: true, Validation: &expr.ValidationExpr{Values: []any{"cat"}}}, "actual"},
		{"inline object", &expr.AttributeExpr{Type: &expr.Object{{Name: "kind", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"cat"}}}}}, Validation: &expr.ValidationExpr{Required: []string{"kind"}}}, "v.Kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope := cg.NewNameScope()
			union := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{{Name: "Branch", Attribute: tc.attribute}}}
			data := buildHTTPUnionTypeData(union, scope, bodyCodecContext(scope, true), false)
			require.Contains(t, data.Fields[0].ValidateCode, "("+tc.operand+` == "cat")`)
			require.NotContains(t, data.Fields[0].ValidateCode, "*"+tc.operand)
		})
	}
}
