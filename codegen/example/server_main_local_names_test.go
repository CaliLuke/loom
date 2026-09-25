package example

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestServerMainLocalNameReservationsCoverGeneratedLocals checks that
// serverMainLocalNames lists every local that the generated example server
// main declares where a service package imported under the same name would
// not compile.
func TestServerMainLocalNameReservationsCoverGeneratedLocals(t *testing.T) {
	const genpkg = "example.com/localnames/gen"
	dir := t.TempDir()
	renderLocalNameServerMain(t, dir, genpkg, "zzsvc")
	for _, name := range testingx.ServicePackageShadowNames(t, dir, genpkg, "zz") {
		if name == "main" {
			// The main function names no importable package.
			continue
		}
		assert.Truef(t, slices.Contains(serverMainLocalNames, name), "generated local %q is missing from serverMainLocalNames", name)
	}
}

// TestServerMainServicePackageNamedLikeLocal checks that the example server
// main imports a service package named like one of its locals under an
// alias and qualifies the service types with it.
func TestServerMainServicePackageNamedLikeLocal(t *testing.T) {
	const genpkg = "example.com/localnames/gen"
	dir := t.TempDir()
	renderLocalNameServerMain(t, dir, genpkg, serverMainLocalNames...)
	code, err := os.ReadFile(filepath.Join(dir, "cmd", "zzapi", "main.go"))
	require.NoError(t, err)
	for _, name := range serverMainLocalNames {
		assert.Contains(t, string(code), name+`svc "`+genpkg+"/"+name+`"`)
		assert.Contains(t, string(code), name+"svc.NewEndpoints(")
		assert.Contains(t, string(code), name+"svc.ServerInterceptors")
	}
}

// renderLocalNameServerMain renders the example server main of a server that
// hosts services with the given names to dir.
func renderLocalNameServerMain(t *testing.T, dir, genpkg string, services ...string) {
	t.Helper()
	root := codegen.RunDSL(t, func() {
		serverMainLocalNameDSL(services...)
	})
	for _, f := range ServerFiles(genpkg, root, service.NewServicesData(root)) {
		_, err := f.Render(dir)
		require.NoError(t, err)
	}
}

// serverMainLocalNameDSL declares an API server that hosts services with the
// given names, each with a server interceptor. Every name it gives to
// generated code contains "zz".
func serverMainLocalNameDSL(services ...string) {
	API("zzapi", func() {
		Server("zzapi", func() {
			Services(services...)
			Host("zzdev", func() {
				URI("http://localhost:8080")
			})
		})
	})
	audit := Interceptor("zzaudit")
	for _, name := range services {
		Service(name, func() {
			ServerInterceptor(audit)
			Method("zzping", func() {
				Payload(func() {
					Attribute("zza", Int)
				})
				HTTP(func() {
					POST("/" + name + "/ping")
				})
			})
		})
	}
}
