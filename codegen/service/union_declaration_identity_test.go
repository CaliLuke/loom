package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestUnionDeclarationsMatchReferencedIdentity(t *testing.T) {
	union := makeUnionForOrderTest("Raw", "text", "number")
	first := &expr.UserTypeExpr{TypeName: "First", AttributeExpr: &expr.AttributeExpr{Type: union}}
	second := &expr.UserTypeExpr{TypeName: "Second", AttributeExpr: &expr.AttributeExpr{Type: union}}
	types := []expr.DataType{union, first, second}
	for _, collector := range []string{"service", "views"} {
		for _, order := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
			t.Run(fmt.Sprintf("%s/%v", collector, order), func(t *testing.T) {
				scope := codegen.NewNameScope()
				var attributes []*expr.AttributeExpr
				loc := &codegen.Location{RelImportPath: "gen/service"}
				for range 2 {
					for _, index := range order {
						attributes = append(attributes, &expr.AttributeExpr{Type: types[index]})
					}
				}
				declarations := collectIdentityTestDeclarations(collector, attributes, scope, loc)
				names := make([]string, 0, len(declarations))
				for _, declaration := range declarations {
					names = append(names, declaration.Name)
				}
				expected := make([]string, 0, len(types))
				for _, typ := range types {
					expected = append(expected, scope.GoTypeName(&expr.AttributeExpr{Type: typ}))
				}
				require.ElementsMatch(t, expected, names)
			})
		}
	}
}

func TestNullableUnionDeclarationIdentity(t *testing.T) {
	for _, collector := range []string{"service", "views"} {
		for _, nullableFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/nullable-first=%t", collector, nullableFirst), func(t *testing.T) {
				union := makeUnionForOrderTest("Raw", "text", "number")
				named := &expr.UserTypeExpr{TypeName: "Named", UID: "named", AttributeExpr: &expr.AttributeExpr{Type: union, Nullable: true}}
				first := &expr.AttributeExpr{Type: named, Nullable: nullableFirst}
				second := &expr.AttributeExpr{Type: named, Nullable: !nullableFirst}
				scope := codegen.NewNameScope()
				declarations := collectIdentityTestDeclarations(collector, []*expr.AttributeExpr{first, second}, scope, nil)
				require.Len(t, declarations, 1)
				declaration := declarations[0]
				require.NotNil(t, declaration)
				require.Equal(t, scope.GoValueTypeName(first), declaration.Name)
				require.Equal(t, "loom.Nullable[Named]", scope.GoTypeRef(first), "declaration identity must not erase nullable use sites")
				require.True(t, named.Nullable)
				require.Equal(t, nullableFirst, first.Nullable)
				require.Equal(t, !nullableFirst, second.Nullable)
			})
		}
	}
}

// Both collectors are exercised in one physical package in these identity tests.
// Keep their distinct map types without flattening the native collector's keys.
func collectIdentityTestDeclarations(collector string, attributes []*expr.AttributeExpr, scope *codegen.NameScope, loc *codegen.Location) []*UnionTypeData {
	var result []*UnionTypeData
	if collector == "service" {
		declarations := make(map[destinationTypeIdentity]*UnionTypeData)
		seen := make(map[destinationTypeIdentity]struct{})
		for _, attribute := range attributes {
			collectUnionTypes(attribute, scope, loc, "", declarations, seen)
		}
		for _, declaration := range declarations {
			result = append(result, declaration)
		}
	} else {
		declarations := make(map[string]*UnionTypeData)
		seen := make(map[string]struct{})
		for _, attribute := range attributes {
			collectViewUnionTypes(attribute, scope, loc, declarations, seen)
		}
		for _, declaration := range declarations {
			result = append(result, declaration)
		}
	}
	return result
}
