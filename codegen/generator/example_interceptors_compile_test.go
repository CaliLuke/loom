package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestExampleInterceptorsCompile generates the service, transport, and example
// output for a service with method-level server and client interceptors, then
// builds and vets the module. The example interceptors must qualify the
// interceptor Info types with the service package, under an alias when its
// name clashes with another import of the example file. The generated service
// code must call the interceptor wrappers it declares whatever the case of the
// interceptor design name.
func TestExampleInterceptorsCompile(t *testing.T) {
	cases := []struct {
		Name string
		// Service is the name of the service.
		Service string
		// PkgPath is the struct:pkg:path of the payload and result type, none
		// when empty.
		PkgPath string
		// Interceptor and GoName are the design and Go names of the
		// interceptor, "Audit" when empty.
		Interceptor, GoName string
		// Import and Qualifier are the import of the service package in the
		// example interceptor files and the qualifier of its types.
		Import, Qualifier string
		// Packages are the packages to build and vet, all when empty.
		Packages []string
	}{
		{
			Name:      "service-package",
			Service:   "shop",
			Import:    `shop "example.com/shop/gen/shop"`,
			Qualifier: "shop",
		},
		{
			Name:      "struct-pkg-path",
			Service:   "shop",
			PkgPath:   "types",
			Import:    `shop "example.com/shop/gen/shop"`,
			Qualifier: "shop",
		},
		{
			Name:      "service-named-log",
			Service:   "log",
			Import:    `log2 "example.com/shop/gen/log"`,
			Qualifier: "log2",
			// The example service stub imports the service package and the
			// clue log package under the same name, so it does not compile
			// (#435).
			Packages: []string{"./gen/...", "./interceptors/..."},
		},
		{
			Name:        "lower-case-interceptor-name",
			Service:     "shop",
			Interceptor: "audit_log",
			GoName:      "AuditLog",
			Import:      `shop "example.com/shop/gen/shop"`,
			Qualifier:   "shop",
		},
	}
	source := loomModuleSource(t)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			name, goName := c.Interceptor, c.GoName
			if name == "" {
				name, goName = "Audit", "Audit"
			}
			codegen.RunDSL(t, func() {
				interceptorExampleDSL(c.Service, c.PkgPath, name)
			})
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/shop\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			for _, side := range []string{"server", "client"} {
				content, err := os.ReadFile(filepath.Join(dir, "interceptors", c.Service+"_"+side+".go"))
				require.NoError(t, err)
				require.Contains(t, string(content), c.Import)
				require.Contains(t, string(content), goName+"(ctx context.Context, info *"+c.Qualifier+"."+goName+"Info, next loom.Endpoint) (any, error)")
			}

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			pkgs := c.Packages
			if len(pkgs) == 0 {
				pkgs = []string{"./..."}
			}
			output, err := testingx.RunCmd(dir, "go", append([]string{"build"}, pkgs...)...)
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", append([]string{"vet"}, pkgs...)...)
			require.NoError(t, err, output)
		})
	}
}

// interceptorExampleDSL declares the service svc with a method "direct" that
// uses the interceptor named interceptor on both the server and the client
// side. The payload and result type is placed in the pkgPath package when
// pkgPath is not empty.
func interceptorExampleDSL(svc, pkgPath, interceptor string) {
	dsl.API("shop", func() {})
	item := dsl.Type("Item", func() {
		if pkgPath != "" {
			dsl.Meta("struct:pkg:path", pkgPath)
		}
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("name", dsl.String)
	})
	audit := dsl.Interceptor(interceptor, func() {
		dsl.ReadPayload(func() {
			dsl.Attribute("id")
		})
		dsl.WriteResult(func() {
			dsl.Attribute("name")
		})
	})
	dsl.Service(svc, func() {
		dsl.Method("direct", func() {
			dsl.ServerInterceptor(audit)
			dsl.ClientInterceptor(audit)
			dsl.Payload(item)
			dsl.Result(item)
			dsl.HTTP(func() {
				dsl.PUT("/v1/direct")
			})
		})
	})
}
