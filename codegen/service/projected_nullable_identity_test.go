package service

import (
	"fmt"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestProjectedNullableDeclarationIdentity(t *testing.T) {
	for _, shape := range []string{"primitive", "object", "union", "array", "map"} {
		for _, intrinsic := range []bool{false, true} {
			for _, nullableFirst := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/intrinsic=%t/nullable-first=%t", shape, intrinsic, nullableFirst), func(t *testing.T) {
					child := projectedNullableIdentityType(shape, intrinsic)
					fields := &expr.Object{
						{Name: "plain", Attribute: &expr.AttributeExpr{Type: child}},
						{Name: "nullable", Attribute: &expr.AttributeExpr{Type: child, Nullable: true}},
					}
					if nullableFirst {
						(*fields)[0], (*fields)[1] = (*fields)[1], (*fields)[0]
					}
					source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Holder", UID: "holder", AttributeExpr: &expr.AttributeExpr{Type: fields}}}
					scope, viewScope := codegen.NewNameScope(), codegen.NewNameScope()
					projected := expr.DupAtt(source)
					types := collectProjectedTypes(projected, source, "views", scope, viewScope, make(map[string]*ProjectedTypeData))
					require.Len(t, types, 2)
					var data *ProjectedTypeData
					for _, candidate := range types {
						if candidate.Type.ID() == child.ID() {
							data = candidate
						}
					}
					require.NotNil(t, data)
					canonical := &expr.AttributeExpr{Type: data.Type}
					expectedName := viewScope.GoValueTypeName(canonical)
					expectedFunction := codegen.NamedValidationFunction(canonical, projectedTypeContext("", !expr.IsPrimitive(data.Type), viewScope), "result")
					t.Run("declaration", func(t *testing.T) {
						require.True(t, token.IsIdentifier(data.VarName), "declaration name must not contain a Nullable carrier: %s", data.VarName)
						require.Equal(t, expectedName, data.VarName)
					})
					require.Len(t, data.Validations, 1)
					t.Run("validator identity", func(t *testing.T) {
						require.Equal(t, expectedFunction.Name, data.Validations[0].Name)
					})
					t.Run("validator carrier", func(t *testing.T) {
						require.Equal(t, expectedFunction.Ref, data.Validations[0].Ref, "first occurrence must not redefine declaration parameter storage")
						require.Equal(t, expectedFunction.Ref, data.Ref, "projected declaration reference must be intrinsic")
					})
					if shape == "union" {
						t.Run("union declaration", func(t *testing.T) {
							unions := make(map[string]*UnionTypeData)
							collectViewUnionTypes(canonical, viewScope, nil, unions, make(map[string]struct{}))
							require.Len(t, unions, 1)
							require.Equal(t, expectedName, unions[data.Type.Hash()].Name)
						})
					}
					require.Equal(t, intrinsic, child.Attribute().Nullable, "projection must not rewrite authored nullability")
					require.True(t, expr.AsObject(source.Type).Attribute("nullable").Nullable)
					require.False(t, expr.AsObject(source.Type).Attribute("plain").Nullable)
				})
			}
		}
	}
}

func TestNullableSelectedViewHelperNames(t *testing.T) {
	child := &expr.ResultTypeExpr{
		UserTypeExpr: &expr.UserTypeExpr{TypeName: "Child", UID: "child", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}},
	}
	childView := &expr.ViewExpr{Name: "tiny", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}, Parent: child}
	child.Views = []*expr.ViewExpr{childView}
	attribute := &expr.AttributeExpr{Type: child, Nullable: true}
	scope := codegen.NewNameScope()
	ctx := projectedTypeContext("", true, scope)
	function := codegen.NamedValidationFunction(&expr.AttributeExpr{Type: child}, ctx, "result")

	t.Run("collection element", func(t *testing.T) {
		collection := &expr.ResultTypeExpr{
			UserTypeExpr: &expr.UserTypeExpr{TypeName: "Children", UID: "children", AttributeExpr: &expr.AttributeExpr{Type: &expr.Array{ElemType: attribute}}},
			Views:        []*expr.ViewExpr{{Name: "tiny"}},
		}
		validations := buildValidations(&expr.AttributeExpr{Type: collection}, scope)
		require.Len(t, validations, 1)
		require.Contains(t, validations[0].Validate, function.Name+"Tiny(item,")
	})
	t.Run("nested selected view", func(t *testing.T) {
		parent := &expr.ResultTypeExpr{
			UserTypeExpr: &expr.UserTypeExpr{TypeName: "Parent", UID: "parent", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: attribute}}}},
		}
		view := &expr.ViewExpr{Name: "default", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: &expr.AttributeExpr{Type: child, Meta: expr.MetaExpr{expr.ViewMetaKey: []string{"tiny"}}}}}}, Parent: parent}
		_, fields := buildViewObjectValidation(&expr.AttributeExpr{Type: parent}, parent, view, scope)
		require.Len(t, fields, 1)
		require.Equal(t, function.Name+"Tiny", fields[0].ValidateVar)
	})
	t.Run("projection helpers", func(t *testing.T) {
		require.True(t, token.IsIdentifier(projectionHelperBaseName(scope, attribute)))
		require.True(t, token.IsIdentifier(projectedResultInitHelperBaseName(scope, scope, attribute, attribute)))
	})
}

func projectedNullableIdentityType(shape string, nullable bool) expr.UserType {
	minimum := 1
	attribute := &expr.AttributeExpr{Nullable: nullable}
	switch shape {
	case "primitive":
		attribute.Type = expr.String
		attribute.Validation = &expr.ValidationExpr{MinLength: &minimum}
	case "object":
		attribute.Type = &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}}
		attribute.Validation = &expr.ValidationExpr{Required: []string{"name"}}
	case "union":
		attribute.Type = &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minimum}}}}}
	case "array":
		attribute.Type = &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
		attribute.Validation = &expr.ValidationExpr{MinLength: &minimum}
	case "map":
		attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}
		attribute.Validation = &expr.ValidationExpr{MinLength: &minimum}
	}
	return &expr.UserTypeExpr{TypeName: "Child", UID: "child", AttributeExpr: attribute}
}
