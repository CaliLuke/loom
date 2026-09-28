package codegen

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestCollectionTypeReferencesMatchDefinitions(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}
	namedUnion := &expr.UserTypeExpr{TypeName: "NamedChoice", UID: "choice", AttributeExpr: &expr.AttributeExpr{Type: union}}
	object := &expr.UserTypeExpr{TypeName: "Record", UID: "record", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}}}
	for _, leaf := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     string
		qualified bool
	}{
		{"union", &expr.AttributeExpr{Type: union}, "Choice", true},
		{"named-union", &expr.AttributeExpr{Type: namedUnion}, "NamedChoice", true},
		{"object", &expr.AttributeExpr{Type: object}, "*Record", true},
		{"string", &expr.AttributeExpr{Type: expr.String}, "string", false},
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes}, "[]byte", false},
		{"any", &expr.AttributeExpr{Type: expr.Any}, "loom.JSONValue", false},
	} {
		for _, nullable := range []bool{false, true} {
			for _, pkg := range []string{"", "svc"} {
				for depth := 1; depth <= 3; depth++ {
					for pattern := range 1 << depth {
						t.Run(fmt.Sprintf("%s/null=%t/pkg=%s/depth=%d/pattern=%d", leaf.name, nullable, pkg, depth, pattern), func(t *testing.T) {
							attribute := *leaf.attribute
							attribute.Nullable = nullable
							value := leaf.value
							pointer := value[0] == '*'
							if pointer {
								value = value[1:]
							}
							if leaf.qualified && pkg != "" {
								value = pkg + "." + value
							}
							if nullable {
								value = "loom.Nullable[" + value + "]"
							} else if pointer {
								value = "*" + value
							}
							current := &attribute
							for level := range depth {
								if pattern&(1<<level) == 0 {
									current = &expr.AttributeExpr{Type: &expr.Array{ElemType: current}}
									value = "[]" + value
								} else {
									current = &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: current}}
									value = "map[string]" + value
								}
							}
							scope := NewNameScope()
							require.Equal(t, value, scope.GoTypeDefWithTargetPkg(current, false, false, pkg))
							require.Equal(t, value, scope.GoFullTypeRef(current, pkg))
							require.Equal(t, value, scope.GoFullTypeRefWithPackages(current, pkg, nil))
							if pkg == "" {
								require.Equal(t, value, scope.GoTypeRef(current))
							}
						})
					}
				}
			}
		}
	}
	// Top-level unions still use pointers; only collection elements use values.
	require.Equal(t, "*svc.Choice", NewNameScope().GoFullTypeRef(&expr.AttributeExpr{Type: union}, "svc"))
	require.Equal(t, "*svc.NamedChoice", NewNameScope().GoFullTypeRef(&expr.AttributeExpr{Type: namedUnion}, "svc"))
}

func TestCollectionUnionReferencesPreservePackageAliases(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}
	choice := &expr.UserTypeExpr{TypeName: "Choice", UID: "located-choice", AttributeExpr: &expr.AttributeExpr{
		Type: union, Meta: expr.MetaExpr{"struct:pkg:path": {"types/log"}},
	}}
	attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Map{
		KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: choice},
	}}}}
	scope := NewNameScopeWithPackageNames(map[string]string{"types/log": "log2"})
	require.Equal(t, "[]map[string]log2.Choice", scope.GoTypeDefWithTargetPkg(attribute, false, false, "svc"))
	require.Equal(t, "[]map[string]log2.Choice", scope.GoFullTypeRef(attribute, "svc"))
	require.Equal(t, "[]map[string]log3.Choice", scope.GoFullTypeRefWithPackages(attribute, "svc", func(*Location) string {
		return "log3"
	}))
}
