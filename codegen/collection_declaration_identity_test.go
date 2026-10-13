package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCollectionDeclarationIdentity(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		reserve  bool
		external bool
	}{
		{"ordinary", "Item", false, false},
		{"collision", "Item", true, false},
		{"reserved", "Use", true, false},
		{"external", "Item", true, true},
	}
	for _, tc := range cases {
		for _, samePackage := range []bool{false, true} {
			contextName := "qualified"
			if samePackage {
				contextName = "same-package"
			}
			t.Run(tc.name+"/"+contextName, func(t *testing.T) {
				scope := NewNameScope()
				if tc.reserve {
					scope.Unique(tc.typeName)
				}
				definition := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
				if tc.external {
					definition.Meta = expr.MetaExpr{"struct:pkg:path": {"types"}}
				}
				attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: tc.typeName, AttributeExpr: definition}}
				declared := scope.GoTypeName(attribute)
				wantName := tc.typeName
				if tc.reserve && !tc.external {
					wantName += "2"
				}
				require.Equal(t, wantName, declared)
				context := NewAttributeContext(false, false, true, "service", scope)
				prefix := "service."
				if samePackage {
					context.SamePackageConversion = true
					prefix = ""
				}
				if tc.external {
					prefix = "types."
				}
				array := &expr.AttributeExpr{Type: &expr.Array{ElemType: attribute}}
				mapped := &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: attribute}}
				shapes := []struct {
					name      string
					attribute *expr.AttributeExpr
					want      string
				}{
					{"element", attribute, "*" + prefix + declared},
					{"array", array, "[]*" + prefix + declared},
					{"map", mapped, "map[string]*" + prefix + declared},
					{"nested", &expr.AttributeExpr{Type: &expr.Array{ElemType: mapped}}, "[]map[string]*" + prefix + declared},
				}
				for _, shape := range shapes {
					t.Run(shape.name, func(t *testing.T) {
						require.Equal(t, shape.want, collectionTypeRef(shape.attribute, context, false))
					})
				}
			})
		}
	}
}

func TestCollectionAllocatedIdentityOrder(t *testing.T) {
	for _, names := range [][]string{{"item", "Item"}, {"Item", "item"}} {
		scope := NewNameScope()
		for _, name := range names {
			attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: expr.String}}}
			declared := scope.GoTypeName(attribute)
			context := NewAttributeContext(false, false, true, "service", scope)
			t.Run(names[0]+"/"+name, func(t *testing.T) {
				require.Equal(t, "service."+declared, collectionTypeRef(attribute, context, true))
			})
		}
	}
}

func TestCollectionNullableValueIdentity(t *testing.T) {
	scope := NewNameScope()
	scope.Unique("Word")
	attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Word", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}, Nullable: true}
	require.Equal(t, "Word2", scope.GoValueTypeName(attribute))
	context := NewAttributeContext(false, false, true, "service", scope)
	require.Equal(t, "loom.Nullable[service.Word2]", collectionTypeRef(attribute, context, false))
}

func TestCollectionExternalAliasIdentity(t *testing.T) {
	scope := NewNameScopeWithPackageNames(map[string]string{"types/log": "logs"})
	scope.Unique("Entry")
	attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Entry", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:pkg:path": {"types/log"}}}}}
	require.Equal(t, "Entry", scope.GoTypeName(attribute))
	context := NewAttributeContext(false, false, true, "service", scope)
	require.Equal(t, "logs.Entry", collectionTypeRef(attribute, context, true))
}
