package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/naming"
)

func TestUserTypePackagesAliasClashingNames(t *testing.T) {
	cases := []struct {
		Name     string
		Reserved []*codegen.ImportSpec
		Imports  []*codegen.ImportSpec
		Loc      *codegen.Location
		Want     []*codegen.ImportSpec
		WantName string
	}{
		{
			Name:     "no-clash",
			Reserved: []*codegen.ImportSpec{{Path: "mime/multipart"}},
			Imports:  []*codegen.ImportSpec{{Name: "common", Path: "example.com/app/gen/common"}},
			Loc:      &codegen.Location{RelImportPath: "common"},
			Want:     []*codegen.ImportSpec{{Name: "common", Path: "example.com/app/gen/common"}},
			WantName: "common",
		},
		{
			Name:     "clash-with-base-name",
			Reserved: []*codegen.ImportSpec{{Path: "mime/multipart"}},
			Imports:  []*codegen.ImportSpec{{Name: "multipart", Path: "example.com/app/gen/types/multipart"}},
			Loc:      &codegen.Location{RelImportPath: "types/multipart"},
			Want:     []*codegen.ImportSpec{{Name: "multipart2", Path: "example.com/app/gen/types/multipart"}},
			WantName: "multipart2",
		},
		{
			Name:     "clash-with-explicit-name",
			Reserved: []*codegen.ImportSpec{codegen.LoomImport("")},
			Imports: []*codegen.ImportSpec{
				{Name: "loom", Path: "example.com/app/gen/types/loom"},
				{Name: "loom", Path: "example.com/app/gen/types/loom"},
			},
			Loc:      &codegen.Location{RelImportPath: "types/loom"},
			Want:     []*codegen.ImportSpec{{Name: "loom2", Path: "example.com/app/gen/types/loom"}},
			WantName: "loom2",
		},
		{
			Name:     "clash-with-escaped-non-ascii-name",
			Reserved: []*codegen.ImportSpec{{Path: "example.com/other/menu00fc"}},
			Imports:  []*codegen.ImportSpec{{Name: "menu00fc", Path: "example.com/app/gen/tipos/menu00fc"}},
			Loc:      &codegen.Location{RelImportPath: naming.EscapeNonASCII("tipos/menü")},
			Want:     []*codegen.ImportSpec{{Name: "menu00fc2", Path: "example.com/app/gen/tipos/menu00fc"}},
			WantName: "menu00fc2",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			scope := codegen.NewNameScope()
			reserveImportNames(scope, c.Reserved)
			pkgs := NewUserTypePackages("example.com/app/gen", scope)
			pkgs.Add(&Data{UserTypeImports: c.Imports})
			assert.Equal(t, c.Want, pkgs.Imports())
			assert.Equal(t, c.WantName, pkgs.PackageName(c.Loc))
		})
	}
}
