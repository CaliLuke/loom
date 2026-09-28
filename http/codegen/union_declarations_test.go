package codegen

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	svc "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

func TestHTTPUnionDeclarationsCoverAllocatedNames(t *testing.T) {
	shared := makeUnionForOrderTest("Choice", "left", "right")
	other := makeUnionForOrderTest("Other", "number")
	types := []*expr.AttributeExpr{
		{Type: shared},
		{Type: &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: shared}}},
		{Type: &expr.UserTypeExpr{TypeName: "Alias", AttributeExpr: &expr.AttributeExpr{Type: shared}}},
		{Type: other},
	}
	orders := unionDeclarationOrders([]int{0, 1, 2, 3})
	for allocationIndex, allocation := range orders {
		t.Run(strconv.Itoa(allocationIndex), func(t *testing.T) {
			for _, collection := range orders {
				scope := cg.NewNameScope()
				for _, index := range allocation {
					scope.GoTypeName(types[index])
				}
				declarations := make(map[string]*svc.UnionTypeData)
				seen := make(map[string]struct{})
				for range 2 {
					for _, index := range collection {
						collectHTTPUnionTypes(types[index], scope, declarations, seen)
					}
				}
				byName := make(map[string]*svc.UnionTypeData)
				for _, declaration := range declarations {
					require.NotContains(t, byName, declaration.Name)
					byName[declaration.Name] = declaration
				}
				require.Len(t, byName, len(types), "allocation %v collection %v", allocation, collection)
				for _, attribute := range types {
					name := scope.GoTypeName(attribute)
					require.Contains(t, byName, name)
					union := expr.AsUnion(attribute.Type)
					require.Len(t, byName[name].Fields, len(union.Values))
					for index, field := range byName[name].Fields {
						require.Equal(t, scope.GoTypeRef(union.Values[index].Attribute), field.FieldType)
					}
				}
			}
		})
	}
}

func unionDeclarationOrders(values []int) [][]int {
	if len(values) == 0 {
		return [][]int{{}}
	}
	var orders [][]int
	for index, value := range values {
		rest := append([]int(nil), values[:index]...)
		rest = append(rest, values[index+1:]...)
		for _, suffix := range unionDeclarationOrders(rest) {
			orders = append(orders, append([]int{value}, suffix...))
		}
	}
	return orders
}
