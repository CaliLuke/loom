package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestConcretePresenceAliases(t *testing.T) {
	for _, depth := range []int{0, 1, 2, 3} {
		for _, nullable := range []bool{false, true} {
			t.Run(fmt.Sprintf("depth=%d/nullable=%t", depth, nullable), func(t *testing.T) {
				child := &expr.AttributeExpr{Type: expr.String, Nullable: true}
				object := &expr.Object{{Name: "child", Attribute: child}}
				source := &expr.AttributeExpr{Type: object, Nullable: nullable}
				for i := 0; i < depth; i++ {
					source = &expr.AttributeExpr{Type: &expr.UserTypeExpr{
						TypeName: fmt.Sprintf("Alias%d", i), AttributeExpr: source,
					}}
				}
				concrete := concretePresenceAttribute(source)
				require.Equal(t, nullable, expr.IsNullable(source), "source contract is unchanged")
				require.False(t, expr.IsNullable(concrete), "all inherited root nullability is consumed")
				require.True(t, expr.IsNullable(expr.AsObject(concrete.Type).Attribute("child")), "child retains its independent carrier")
				original, copied := source, concrete
				for i := 0; i < depth; i++ {
					originalType := original.Type.(expr.UserType)
					copiedType := copied.Type.(expr.UserType)
					require.Equal(t, originalType.ID(), copiedType.ID())
					original, copied = originalType.Attribute(), copiedType.Attribute()
				}
			})
		}
	}
}

func TestConcretePresenceRecursiveChild(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node"}
	node.AttributeExpr = &expr.AttributeExpr{Type: &expr.Object{
		{Name: "next", Attribute: &expr.AttributeExpr{Type: node}},
	}, Nullable: true}
	alias := &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: node}}
	source := &expr.AttributeExpr{Type: alias}
	concrete := concretePresenceAttribute(source)
	require.False(t, expr.IsNullable(concrete))
	require.True(t, expr.IsNullable(source))
	child := expr.AsObject(concrete.Type).Attribute("next")
	require.True(t, expr.IsNullable(child), "recursive child still targets the nullable declaration")
	require.Equal(t, node.ID(), child.Type.(expr.UserType).ID())
}

func TestConcretePresenceAliasConsumers(t *testing.T) {
	minimum := 2
	for _, kind := range []string{"scalar", "array", "map"} {
		t.Run(kind, func(t *testing.T) {
			leaf := &expr.AttributeExpr{Type: expr.String, Nullable: true,
				Validation: &expr.ValidationExpr{MinLength: &minimum}}
			var underlying expr.DataType = expr.String
			switch kind {
			case "array":
				underlying = &expr.Array{ElemType: leaf}
			case "map":
				underlying = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: leaf}
			}
			base := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{
				Type: underlying, Nullable: true, Validation: &expr.ValidationExpr{MinLength: &minimum},
			}}
			alias := &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: base}}
			source := &expr.AttributeExpr{Type: base}
			target := &expr.AttributeExpr{Type: alias}
			concrete := concretePresenceAttribute(target)
			require.False(t, expr.IsNullable(concrete))
			require.True(t, expr.IsNullable(target))
			switch kind {
			case "array":
				require.True(t, expr.IsNullable(expr.AsArray(concrete.Type).ElemType))
			case "map":
				require.True(t, expr.IsNullable(expr.AsMap(concrete.Type).ElemType))
			}
			ctx := NewAttributeContext(false, false, true, "", NewNameScope())
			for _, pair := range [][2]*expr.AttributeExpr{{source, target}, {target, source}} {
				codePlan, err := GoTransform(pair[0], pair[1], "source", "target", ctx, ctx, "convert", true)
				code := codePlan.Code
				require.NoError(t, err)
				require.Contains(t, code, "source.Value()")
			}
			validation := ValidationCode(target, alias, ctx, true, false, false, "body")
			require.Contains(t, validation, "body.Value()")
			require.NotContains(t, validation, "actual.Value()", "the root carrier is extracted once")
		})
	}
}

func TestConcretePresenceAliasCycle(t *testing.T) {
	first := &expr.UserTypeExpr{TypeName: "First"}
	second := &expr.UserTypeExpr{TypeName: "Second"}
	first.AttributeExpr = &expr.AttributeExpr{Type: second, Nullable: true}
	second.AttributeExpr = &expr.AttributeExpr{Type: first}
	source := &expr.AttributeExpr{Type: first}
	concrete := concretePresenceAttribute(source)
	require.False(t, expr.IsNullable(concrete))
	require.True(t, expr.IsNullable(source))
	require.NotSame(t, first, concrete.Type)
}
