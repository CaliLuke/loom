package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestUnionTransformNullableBranchDispatch(t *testing.T) {
	for _, shape := range []string{"object", "array", "map", "union", "primitive"} {
		for _, mode := range []string{"ordinary", "intrinsic", "occurrence"} {
			t.Run(fmt.Sprintf("%s/%s", shape, mode), func(t *testing.T) {
				source := nullableTransformUnion(shape, "Source", mode)
				target := nullableTransformUnion(shape, "Target", mode)
				scope := NewNameScope()
				ctx := NewAttributeContext(false, false, true, "", scope)
				code, helpers, err := GoTransform(source, target, "source", "target", ctx, ctx, "convert", true)
				require.NoError(t, err)
				if mode == "ordinary" {
					require.NotContains(t, code, "actual.IsNull()")
					if shape == "object" {
						require.Len(t, helpers, 1)
						require.Contains(t, code, helpers[0].Name+"(actual)")
					}
					return
				}
				require.Contains(t, code, "actual.IsNull()", "a selected Nullable carrier must reach the shared presence dispatcher")
				require.Contains(t, code, "actual.Value()")
				if shape == "object" {
					require.Len(t, helpers, 1)
					require.Contains(t, code, helpers[0].Name+"(&actual)", "union calls must use the declared concrete helper with an unwrapped object")
					require.Equal(t, "*SourceBranch", helpers[0].ParamTypeRef)
					require.Equal(t, "*TargetBranch", helpers[0].ResultTypeRef)
				}
			})
		}
	}
}

func TestUnionTransformRejectsNullableMismatch(t *testing.T) {
	for _, modes := range [][2]string{{"intrinsic", "ordinary"}, {"ordinary", "occurrence"}} {
		t.Run(modes[0]+" to "+modes[1], func(t *testing.T) {
			source := nullableTransformUnion("object", "Source", modes[0])
			target := nullableTransformUnion("object", "Target", modes[1])
			scope := NewNameScope()
			ctx := NewAttributeContext(false, false, true, "", scope)
			_, _, err := GoTransform(source, target, "source", "target", ctx, ctx, "convert", true)
			require.ErrorContains(t, err, "cannot transform nullable attribute to a non-nullable representation")
		})
	}
}

func nullableTransformUnion(shape, prefix, mode string) *expr.AttributeExpr {
	attribute := &expr.AttributeExpr{Nullable: mode == "intrinsic"}
	switch shape {
	case "object":
		attribute.Type = &expr.Object{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}
	case "array":
		attribute.Type = &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
	case "map":
		attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}
	case "union":
		attribute.Type = &expr.Union{TypeName: prefix + "Nested", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
	case "primitive":
		attribute.Type = expr.String
	}
	branch := &expr.AttributeExpr{
		Type:     &expr.UserTypeExpr{TypeName: prefix + "Branch", UID: prefix + "-branch", AttributeExpr: attribute},
		Nullable: mode == "occurrence",
	}
	return &expr.AttributeExpr{Type: &expr.Union{TypeName: prefix + "Union", Values: []*expr.NamedAttributeExpr{{Name: "branch", Attribute: branch}}}}
}
