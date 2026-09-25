package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"

	"github.com/CaliLuke/loom/expr"
)

func TestUserTypeLocationEscapesNonASCII(t *testing.T) {
	cases := []struct {
		Name       string
		Type       string
		PkgPath    string
		ImportPath string
		FilePath   string
		Package    string
	}{
		{"ascii", "Menu", "shared/types", "shared/types", filepath.Join("shared", "types", "menu.go"), "types"},
		{"ascii dotted", "Menu", "types.v2", "types.v2", filepath.Join("types.v2", "menu.go"), "typesV2"},
		{"latin accent", "Menü", "tipos/menü", "tipos/menu00fc", filepath.Join("tipos", "menu00fc", "menü.go"), "menu00fc"},
		{"cjk", "Menu", "型", "u578b", filepath.Join("u578b", "menu.go"), "u578b"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ut := &expr.UserTypeExpr{
				TypeName:      c.Type,
				AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}, Meta: expr.MetaExpr{"struct:pkg:path": {c.PkgPath}}},
			}
			loc := UserTypeLocation(ut)
			if assert.NotNil(t, loc) {
				assert.Equal(t, c.ImportPath, loc.RelImportPath)
				assert.Equal(t, c.FilePath, loc.FilePath)
				assert.Equal(t, c.Package, loc.PackageName())
				assert.NoError(t, module.CheckImportPath("example.com/gen/"+loc.RelImportPath))
			}
			att := &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:pkg:path": {c.PkgPath}}}
			assert.Equal(t, c.Package, NewNameScope().attributePkgName(att))
		})
	}
}
