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
