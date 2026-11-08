package codegen

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
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
				declarations := make(map[string]*uniongen.Type)
				seen := make(map[string]struct{})
				for range 2 {
					for _, index := range collection {
						collectHTTPUnionTypes(types[index], scope, bodyCodecContext(scope, false), declarations, seen, false, declarations)
					}
				}
				byName := make(map[string]*uniongen.Type)
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

func TestHTTPNullableUnionDeclarationIdentity(t *testing.T) {
	for _, closedObjects := range []bool{false, true} {
		for _, nullableFirst := range []bool{false, true} {
			t.Run(strconv.FormatBool(closedObjects)+"/"+strconv.FormatBool(nullableFirst), func(t *testing.T) {
				union := makeUnionForOrderTest("Raw", "text", "number")
				named := &expr.UserTypeExpr{TypeName: "Named", UID: "named", AttributeExpr: &expr.AttributeExpr{Type: union, Nullable: true}}
				first := &expr.AttributeExpr{Type: named, Nullable: nullableFirst}
				second := &expr.AttributeExpr{Type: named, Nullable: !nullableFirst}
				scope := cg.NewNameScope()
				allocated := make(map[string]*uniongen.Type)
				client, server := make(map[string]*uniongen.Type), make(map[string]*uniongen.Type)
				collectHTTPUnionTypes(first, scope, bodyCodecContext(scope, false), client, make(map[string]struct{}), closedObjects, allocated)
				collectHTTPUnionTypes(second, scope, bodyCodecContext(scope, false), server, make(map[string]struct{}), closedObjects, allocated)
				name := scope.GoValueTypeName(first)
				require.Len(t, allocated, 1)
				require.Contains(t, allocated, name)
				require.Equal(t, allocated[name].Name, client[name].Name)
				require.Equal(t, allocated[name].KindName, client[name].KindName)
				require.Equal(t, allocated[name].Name, server[name].Name)
				require.Equal(t, allocated[name].KindName, server[name].KindName)
				require.NotSame(t, client[name], server[name])
				require.Equal(t, name, allocated[name].Name)
				require.Equal(t, "loom.Nullable[Named]", scope.GoTypeRef(first))
				require.True(t, named.Nullable)
				require.Equal(t, nullableFirst, first.Nullable)
				require.Equal(t, !nullableFirst, second.Nullable)
			})
		}
	}
}

func TestHTTPUnionFormBindingsBelongToPhysicalPackage(t *testing.T) {
	for _, order := range [][]bool{{true, false}, {false, true}} {
		scope := cg.NewNameScope()
		child := &expr.UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
			{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}}}
		union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "Record", Attribute: &expr.AttributeExpr{Type: child}}}}
		attribute := &expr.AttributeExpr{Type: union}
		allocated := make(map[string]*uniongen.Type)
		published := make(map[bool]*uniongen.Type)
		for _, presence := range order {
			context := bodyCodecContext(scope, false)
			context.JSONPresence = presence
			context.JSONPresenceTypes = map[string]bool{child.Hash(): presence}
			context.PresencePointerTypes = map[string]bool{child.Hash(): !presence}
			context.PresenceUseDefaultTypes = map[string]bool{child.Hash(): presence}
			members := make(map[string]*uniongen.Type)
			collectHTTPUnionTypes(attribute, scope, context, members, make(map[string]struct{}), false, allocated)
			published[presence] = members[scope.GoTypeName(attribute)]
		}
		require.Len(t, allocated, 1)
		require.Equal(t, published[true].Name, published[false].Name)
		require.Equal(t, published[true].KindName, published[false].KindName)
		require.Equal(t, published[true].Fields[0].FieldType, published[false].Fields[0].FieldType)
		require.Equal(t, "value.Value.Present()", published[true].Fields[0].FormFields[0].Presence)
		require.Empty(t, published[false].Fields[0].FormFields[0].Presence)
		require.NotSame(t, published[true].Fields[0], published[false].Fields[0])
	}
}
