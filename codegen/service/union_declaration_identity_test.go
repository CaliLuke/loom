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
	for _, collector := range []struct {
		name    string
		collect func(*expr.AttributeExpr, *codegen.NameScope, *codegen.Location, map[string]*UnionTypeData, map[string]struct{})
	}{
		{"service", collectUnionTypes},
		{"views", collectViewUnionTypes},
	} {
		for _, order := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
			t.Run(fmt.Sprintf("%s/%v", collector.name, order), func(t *testing.T) {
				scope := codegen.NewNameScope()
				declarations := make(map[string]*UnionTypeData)
				seen := make(map[string]struct{})
				loc := &codegen.Location{RelImportPath: "gen/service"}
				for range 2 {
					for _, index := range order {
						collector.collect(&expr.AttributeExpr{Type: types[index]}, scope, loc, declarations, seen)
					}
				}
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
	for _, collector := range []struct {
		name    string
		collect func(*expr.AttributeExpr, *codegen.NameScope, *codegen.Location, map[string]*UnionTypeData, map[string]struct{})
	}{
		{"service", collectUnionTypes},
		{"views", collectViewUnionTypes},
	} {
		for _, nullableFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/nullable-first=%t", collector.name, nullableFirst), func(t *testing.T) {
				union := makeUnionForOrderTest("Raw", "text", "number")
				named := &expr.UserTypeExpr{TypeName: "Named", UID: "named", AttributeExpr: &expr.AttributeExpr{Type: union, Nullable: true}}
				first := &expr.AttributeExpr{Type: named, Nullable: nullableFirst}
				second := &expr.AttributeExpr{Type: named, Nullable: !nullableFirst}
				scope := codegen.NewNameScope()
				declarations := make(map[string]*UnionTypeData)
				seen := make(map[string]struct{})
				collector.collect(first, scope, nil, declarations, seen)
				collector.collect(second, scope, nil, declarations, seen)
				require.Len(t, declarations, 1)
				declaration := declarations[named.Hash()]
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
