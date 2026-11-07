package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNameScopeDestinationIdentity(t *testing.T) {
	common := &Location{RelImportPath: "common"}
	other := &Location{RelImportPath: "other"}
	union := &expr.Union{TypeName: "Pick"}
	user := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:pkg:path": {"common"}}}}
	scope := NewNameScopeWithPackageNames(map[string]string{"common": "shared"})
	scope.SetTypeNames(common, func(typ expr.DataType, location *Location) (string, bool) {
		if location.RelImportPath == "other" {
			return typ.Name() + "3", true
		}
		return typ.Name() + "2", true
	})
	for _, view := range []*NameScope{scope, scope.WithoutPackageNames(), NewNameScopeLike(scope)} {
		require.Equal(t, "Choice2", view.GoValueTypeName(&expr.AttributeExpr{Type: user}))
		require.Equal(t, "alias.Choice2", view.GoFullTypeName(&expr.AttributeExpr{Type: user}, "alias"))
		require.Equal(t, "Pick2", view.GoValueTypeName(&expr.AttributeExpr{Type: union}))
		require.Equal(t, "Pick3", view.WithTypeNameLocation(other).GoValueTypeName(&expr.AttributeExpr{Type: union}))
		require.Equal(t, "Choice2", view.WithTypeNameLocation(other).GoValueTypeName(&expr.AttributeExpr{Type: user}), "explicit user destination wins")
		require.Equal(t, "Choice2", view.PeekUnique("Choice2"), "external declarations do not occupy a consumer's namespace")
		primitive := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"opaque.Token", "example.com/opaque"}}}
		require.Equal(t, "opaque.Token", view.GoValueTypeName(primitive))
	}
}
