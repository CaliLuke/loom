package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

func TestServicePackageNames(t *testing.T) {
	cases := []struct {
		Name         string
		Service      string
		ExpectedPath string
		ExpectedPkg  string
	}{
		{"init", "init", "init", "init_"},
		{"init title case", "Init", "init", "init_"},
		{"init upper case", "INIT", "init", "init_"},
		{"ascii", "calc service", "calc_service", "calcservice"},
		{"ascii acronym", "MyAPIService", "my_api_service", "myapiservice"},
		{"latin accent", "Café", "cafu00e9", "cafu00e9"},
		{"cjk", "日本語", "u65e5u672cu8a9e", "u65e5u672cu8a9e"},
		{"astral letter", "𝒜lpha", "U0001d49clpha", "U0001d49clpha"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			path := naming.ServiceDir(c.Service)
			assert.Equal(t, c.ExpectedPath, path)
			assert.NoError(t, module.CheckImportPath("example.com/gen/"+path))
			assert.Equal(t, c.ExpectedPkg, PackageBaseName(c.Service))
		})
	}
}

func TestServicePackageNameAvoidsMainPackage(t *testing.T) {
	service := &expr.ServiceExpr{Name: "main"}

	assert.Equal(t, "mainsvc", PackageName(nil, service))
}

// TestServicePackageInitIdentity checks that declarations and reference queries
// share a legal identity, and that existing allocation still owns collisions.
func TestServicePackageInitIdentity(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("demo", func() {
		})
		dsl.Service("init", func() {
			dsl.Method("echo", func() {
				dsl.Result(dsl.String)
			})
		})
	})
	service := root.Services[0]
	data := NewServicesData(root).Get(service.Name)
	assert.Equal(t, "init_", PackageName(root, service))
	assert.Equal(t, PackageName(root, service), data.PkgName)
	assert.Equal(t, "init", data.PathName)
	assert.Equal(t, "init", service.Name)
	scope := newServiceNameScope(nil)
	scope.Unique("init_")
	assert.Equal(t, "init_svc", servicePackageName(scope, root, service))
	assert.Equal(t, "init_svc", servicePackageName(scope, root, service))
}
