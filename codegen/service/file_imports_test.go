package service

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestFileImportsAliasesClashingPackages checks that a file imports a
// struct:pkg:path package under an alias only when its name clashes with
// another import of the file, and that the data of the file qualifies the
// types of the package with the alias.
func TestFileImportsAliasesClashingPackages(t *testing.T) {
	const genpkg = "example.com/app/gen"
	root := codegen.RunDSL(t, fileImportsDSL)
	services := NewServicesData(root)
	data := services.Get("svc")
	security := &codegen.ImportSpec{Name: "security", Path: genpkg + "/types/security"}
	loomSecurity := codegen.LoomImport("security")
	context := codegen.SimpleImport("context")

	names, specs := data.FileImports([]*codegen.ImportSpec{context, loomSecurity, security})
	assert.Nil(t, names, "no package is renamed before SetUserTypeImports")
	assert.Equal(t, []*codegen.ImportSpec{context, loomSecurity, security}, specs)

	require.NoError(t, SetUserTypeImports(genpkg, data))
	require.Equal(t, []*codegen.ImportSpec{security}, data.UserTypeImports)

	imports := []*codegen.ImportSpec{context, security}
	names, specs = data.FileImports(imports)
	assert.Nil(t, names)
	assert.Same(t, &imports[0], &specs[0], "the imports are returned as is when no name clashes")

	names, specs = data.FileImports([]*codegen.ImportSpec{context, loomSecurity, security})
	assert.Equal(t, map[string]string{"types/security": "security2"}, names)
	assert.Equal(t, []*codegen.ImportSpec{context, loomSecurity, {Name: "security2", Path: security.Path}}, specs)
	again, specsAgain := data.FileImports(specs)
	assert.Equal(t, names, again, "FileImports is idempotent")
	assert.Equal(t, specs, specsAgain)

	assert.Same(t, services, services.WithPackageNames(nil))
	renamed := services.WithPackageNames(names)
	require.NotSame(t, services, renamed)
	assert.Same(t, renamed, services.WithPackageNames(map[string]string{"types/security": "security2"}))
	renamedData := renamed.Get("svc")
	assert.Equal(t, data.UserTypeImports, renamedData.UserTypeImports)
	cases := []struct {
		Method, Ref, RenamedRef, Name string
	}{
		{Method: "show", Ref: "*security.Item", RenamedRef: "*security2.Item", Name: "Item"},
		{Method: "list", Ref: "[]*security.Item", RenamedRef: "[]*security2.Item", Name: "[]*security.Item"},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			assert.Equal(t, c.Ref, data.Method(c.Method).ResultRef)
			assert.Equal(t, c.RenamedRef, renamedData.Method(c.Method).ResultRef)
			assert.Equal(t, c.Name, data.Method(c.Method).Result)
			assert.Equal(t, c.Name, renamedData.Method(c.Method).Result, "names do not depend on the package aliases")
		})
	}
}

// TestServiceFilesAliasClashingPackages checks that the service, endpoints and
// client files import a struct:pkg:path package named like the Loom security
// package under an alias when they import both, and under its name otherwise.
func TestServiceFilesAliasClashingPackages(t *testing.T) {
	const genpkg = "example.com/app/gen"
	root := codegen.RunDSL(t, fileImportsDSL)
	services := NewServicesData(root)
	require.NoError(t, SetUserTypeImports(genpkg, services.Get("svc")))
	files := Files(genpkg, root.Service("svc"), services, make(map[string][]string))
	files = append(files, EndpointFile(genpkg, root.Service("svc"), services), ClientFile(genpkg, root.Service("svc"), services))
	cases := []struct {
		Path   string
		Import codegen.ImportSpec
		Code   string
	}{
		{Path: "service.go", Import: codegen.ImportSpec{Name: "security2", Path: genpkg + "/types/security"}, Code: "Show(context.Context) (res *security2.Item, err error)"},
		{Path: "endpoints.go", Import: codegen.ImportSpec{Name: "security2", Path: genpkg + "/types/security"}, Code: "security.AuthJWTFunc"},
		{Path: "client.go", Import: codegen.ImportSpec{Name: "security", Path: genpkg + "/types/security"}, Code: "res.(*security.Item)"},
	}
	for _, c := range cases {
		t.Run(c.Path, func(t *testing.T) {
			var file *codegen.File
			for _, f := range files {
				if f.Path == filepath.Join(codegen.Gendir, "svc", c.Path) {
					file = f
				}
			}
			require.NotNil(t, file)
			header := codegen.HeaderDataForSection(file.HeaderSection())
			assert.Contains(t, importValues(header.Imports), c.Import)
			assert.Contains(t, codegen.SectionsCode(t, file.AllSections()[1:]), c.Code)
		})
	}
}

// importValues returns the values of specs.
func importValues(specs []*codegen.ImportSpec) []codegen.ImportSpec {
	values := make([]codegen.ImportSpec, len(specs))
	for i, spec := range specs {
		values[i] = *spec
	}
	return values
}

// fileImportsDSL declares a service secured with JWT whose methods return the
// "Item" type of the struct:pkg:path package "types/security".
func fileImportsDSL() {
	item := dsl.Type("Item", func() {
		dsl.Meta("struct:pkg:path", "types/security")
		dsl.Attribute("id", dsl.String)
	})
	jwt := dsl.JWTSecurity("jwt")
	dsl.Service("svc", func() {
		dsl.Method("show", func() {
			dsl.Result(item)
			dsl.HTTP(func() {
				dsl.GET("/show")
			})
		})
		dsl.Method("list", func() {
			dsl.Security(jwt)
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
			})
			dsl.Result(dsl.ArrayOf(item))
			dsl.HTTP(func() {
				dsl.GET("/list")
			})
		})
	})
}
