package codegen

import (
	"fmt"
	"go/format"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestValidationFunctionContext(t *testing.T) {
	for _, argument := range []string{"body", "validationPath", "validationContext", "validationPart"} {
		t.Run(argument, func(t *testing.T) {
			function := ValidationFunction{Name: "ValidateNode", Argument: argument, Root: "authored-root", Ref: "*Node"}
			base := NewAttributeContext(true, false, false, "", NewNameScope())
			base.ValidationPrefix = "validateAccess"
			ctx := function.Context(base)
			require.Empty(t, base.validationRoot)
			require.True(t, ctx.Pointer)
			require.Equal(t, "validateAccess", ctx.ValidationPrefix)
			for _, suffix := range []string{"", ".child", "[*].child", "[key].child"} {
				require.Equal(t, function.Path(suffix), validationContextExpr(ctx, "authored-root"+suffix))
			}
			require.Equal(t, `"unrelated.child"`, validationContextExpr(ctx, "unrelated.child"))
			require.NotEqual(t, argument, function.Path(""))
			body := "err = check(" + function.Path(".child") + ")"
			definition := function.Definition(body)
			_, err := format.Source([]byte("package generated\n" + definition))
			require.NoError(t, err)
			require.Contains(t, definition, `:= "authored-root"`)
			require.Equal(t, "ValidateNode(value)", function.Call("value", ""))
			require.Equal(t, "ValidateNode(value, "+function.Path(".child")+")", function.Call("value", function.Path(".child")))
		})
	}
}

func TestRecursiveValidationDiagnosticContext(t *testing.T) {
	for _, presence := range []string{"native", "optional", "nullable"} {
		t.Run(presence, func(t *testing.T) {
			node, _ := recursiveValidationReadinessGraph("self", false, presence == "nullable")
			base := NewAttributeContext(true, false, false, "", NewNameScope())
			base.JSONPresence = presence == "optional"
			base.ValidationPrefix = "validateAccess"
			function := ValidationFunction{Name: "validateAccessNode", Argument: "value", Ref: "*Node"}
			ctx := function.Context(base)
			code := ValidationCode(node.Attribute(), node, ctx, true, false, false, "value")
			require.Contains(t, code, `validationPath + ".value"`)
			require.Contains(t, code, `validationPath + ".next"`)
			require.Equal(t, 1, strings.Count(code, "validateAccessNode("))
			require.NotContains(t, code, `"value.next"`)
			if presence != "native" {
				require.Contains(t, code, "validateAccessNode(&actual,")
				require.NotContains(t, code, "actual != nil")
			}
			// Named field layout is retained independently from the unwrapped
			// object value passed to the complete validator.
			concrete := presenceValueContext(ctx, &expr.AttributeExpr{Type: node})
			require.True(t, concrete.Pointer)
			require.True(t, concrete.validationValue)
		})
	}
}

func TestNamedNullableValidationCallCarrier(t *testing.T) {
	for _, intrinsic := range []bool{false, true} {
		t.Run(fmt.Sprintf("intrinsic=%t", intrinsic), func(t *testing.T) {
			node := &expr.UserTypeExpr{TypeName: "Node", UID: "Node", AttributeExpr: &expr.AttributeExpr{
				Nullable: intrinsic,
				Type:     &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Pattern: "^[a-z]+$"}}}},
			}}
			field := &expr.AttributeExpr{Type: node, Nullable: true, Validation: &expr.ValidationExpr{Required: []string{"value"}}}
			parent := &expr.AttributeExpr{Type: &expr.Object{{Name: "node", Attribute: field}}}
			ctx := NewAttributeContext(true, false, false, "", NewNameScope())
			ctx.ValidationPrefix = "validateAccess"
			code := ValidationCode(parent, nil, ctx, true, false, false, "body")
			require.Contains(t, code, "actual.Value == nil")
			function := NamedValidationFunction(&expr.AttributeExpr{Type: node}, ctx, "body")
			require.Equal(t, "validateAccessNode", function.Name)
			if intrinsic {
				require.Equal(t, "loom.Nullable[Node]", function.Ref)
			} else {
				require.Equal(t, "*Node", function.Ref)
			}
			ctx.PresencePointerTypes = map[string]bool{node.Hash(): false}
			resolved := NamedValidationFunction(&expr.AttributeExpr{Type: node}, ctx, "body").Context(ctx)
			require.False(t, resolved.Pointer)
			require.False(t, resolved.validationValue)
			if intrinsic {
				require.Contains(t, code, `validateAccessNode(body.Node, "body.node")`)
				require.NotContains(t, code, "validateAccessNode(&actual")
			} else {
				require.Contains(t, code, `validateAccessNode(&actual, "body.node")`)
			}
		})
	}
}
