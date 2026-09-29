package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

func TestServerValidationUsesGeneratedTypeIdentity(t *testing.T) {
	for _, c := range []struct {
		name       string
		bodyName   string
		identifier string
		want       bool
	}{
		{"different bodies, shared design", "ResponseBody", "application/vnd.shared", false},
		{"same body, distinct designs", "RequestBody", "application/vnd.other", true},
		{"same body and design", "RequestBody", "application/vnd.shared", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			request := identityResultType("RequestBody", expr.String)
			response := identityResultType(c.bodyName, expr.String)
			response.Identifier = c.identifier
			data := &ServiceData{}
			recordServerRequestValidationTypes(data, &expr.AttributeExpr{Type: request})
			require.True(t, shouldGenerateAttributeValidation(request, false, true, data))
			require.Equal(t, c.want, shouldGenerateAttributeValidation(response, false, true, data))
		})
	}
}

func TestCollectUserTypesKeepsDistinctBodyVariants(t *testing.T) {
	first := identityResultType("FirstBody", expr.String)
	second := identityResultType("SecondBody", expr.Int)
	for _, order := range [][]expr.UserType{{first, second}, {second, first}} {
		root := &expr.Object{
			{Name: "first", Attribute: &expr.AttributeExpr{Type: order[0]}},
			{Name: "second", Attribute: &expr.AttributeExpr{Type: order[1]}},
		}
		var names []string
		representation.WalkUserTypes(root, func(userType expr.UserType) {
			names = append(names, userType.Name())
		})
		require.ElementsMatch(t, []string{"FirstBody", "SecondBody"}, names)
	}
}

func TestContainsUnionKeepsDistinctBodyVariants(t *testing.T) {
	plain := identityResultType("PlainBody", expr.String)
	union := identityResultType("UnionBody", makeUnionForOrderTest("Choice", "text", "number"))
	for _, order := range [][]expr.UserType{{plain, union}, {union, plain}} {
		root := &expr.Object{
			{Name: "first", Attribute: &expr.AttributeExpr{Type: order[0]}},
			{Name: "second", Attribute: &expr.AttributeExpr{Type: order[1]}},
		}
		require.True(t, containsUnionType(root))
	}
}

func TestMultipartBodyVariantCycleDetection(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		inner := identityResultType("InnerBody", &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}})
		outer := identityResultType("OuterBody", &expr.Object{{Name: "child", Attribute: &expr.AttributeExpr{Type: inner}}})
		if cycle {
			expr.AsObject(inner.Type).Set("parent", &expr.AttributeExpr{Type: outer})
		}
		require.Equal(t, !cycle, supportsGeneratedMultipartNested(&expr.AttributeExpr{Type: outer}, make(map[string]struct{})))
	}
}

func TestUnionBranchValidatorsUseGeneratedTypeIdentity(t *testing.T) {
	first := identityResultType("FirstBody", expr.String)
	second := identityResultType("SecondBody", expr.String)
	union := &expr.AttributeExpr{Type: &expr.Union{
		TypeName: "Choice",
		Values: []*expr.NamedAttributeExpr{
			{Name: "first", Attribute: &expr.AttributeExpr{Type: first}},
			{Name: "second", Attribute: &expr.AttributeExpr{Type: second}},
		},
	}}
	types := make(map[string]struct{})
	collectUnionBranchUserTypes(union, types)
	require.Len(t, types, 2)
	require.Contains(t, types, first.Hash())
	require.Contains(t, types, second.Hash())
}

func TestGeneratedTypeTraversalTerminatesOnRecursion(t *testing.T) {
	first := identityResultType("FirstBody", expr.String)
	second := identityResultType("SecondBody", makeUnionForOrderTest("Choice", "text", "number"))
	expr.AsObject(first.Type).Set("child", &expr.AttributeExpr{Type: second})
	expr.AsObject(second.Type).Set("parent", &expr.AttributeExpr{Type: first})
	var names []string
	representation.WalkUserTypes(first, func(userType expr.UserType) {
		names = append(names, userType.Name())
	})
	require.ElementsMatch(t, []string{"FirstBody", "SecondBody"}, names)
	require.True(t, containsUnionType(first))
}

func identityResultType(name string, value expr.DataType) *expr.ResultTypeExpr {
	if !expr.IsObject(value) {
		value = &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: value}}}
	}
	return &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: value}},
		Identifier:   "application/vnd.shared",
	}
}
