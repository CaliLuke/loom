package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAttributeContextPhysicalLocations(t *testing.T) {
	scope := NewNameScopeWithPackageNames(map[string]string{"left": "leftAlias", "right": "rightAlias"})
	scope.SetTypeNames(&Location{RelImportPath: "service"}, func(typ expr.DataType, location *Location) (string, bool) {
		if location.RelImportPath == "left" {
			return typ.Name() + "2", true
		}
		if location.RelImportPath == "right" {
			return typ.Name() + "3", true
		}
		return "", false
	})
	parent := func(path string) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Holder", AttributeExpr: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:pkg:path": {path}}}}}
	}
	base := NewAttributeContext(false, false, true, "service", scope)
	source := base.ForAttribute(parent("left"))
	target := base.ForAttribute(parent("right"))
	raw := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Pick"}}
	require.Equal(t, "leftAlias.Pick2", source.Scope.Name(raw, source.Pkg(raw), false, false))
	require.Equal(t, "rightAlias.Pick3", target.Scope.Name(raw, target.Pkg(raw), false, false))
	require.Equal(t, "left", source.NamePkg(raw))
	require.Equal(t, "right", target.NamePkg(raw))
	require.Equal(t, "service", base.DefaultPkg)
	require.Equal(t, "service", base.Scope.Scope().typeLocation.RelImportPath)
	require.Same(t, source, source.ForAttribute(raw), "bare children inherit the enclosing destination")
	require.Equal(t, "leftAlias.Pick2", source.Scope.Name(raw, source.Pkg(raw), false, false), "target derivation must not mutate source")
	custom := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"opaque.Token", "example.com/opaque"}}}
	require.Equal(t, "opaque.Token", target.Scope.Ref(custom, target.Pkg(custom)))
	local := NewAttributeContext(false, false, true, "", source.Scope.Scope())
	require.Same(t, local, local.ForAttribute(parent("left")), "already local declarations retain unqualified output spelling")
}
