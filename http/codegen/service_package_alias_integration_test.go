package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

var httpServicePackageCollisionNames = []string{"strconv", "body", "bodysvc"}

func TestHTTPServicePackageAliasAnalysis(t *testing.T) {
	root := RunHTTPDSL(t, httpServicePackageAliasDSL)
	services := CreateHTTPServices(root)

	require.Equal(t, "ordinary", services.ServicesData.Get("ordinary").PkgName)
	require.Equal(t, "ordinary", services.Get("ordinary").Service.PkgName)
	wantAliases := map[string]string{
		"strconv": "strconvsvc",
		"body":    "bodysvc",
		"bodysvc": "bodysvcsvc",
	}
	for _, name := range httpServicePackageCollisionNames {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, name, services.ServicesData.Get(name).PkgName)
			require.Equal(t, wantAliases[name], services.Get(name).Service.PkgName)
			require.Equal(t, wantAliases[name], services.Get(name).Endpoints[0].ServicePkgName)
		})
	}
}

func TestHTTPServicePackageAliasGeneratedModuleCompiles(t *testing.T) {
	const modulePath = "example.com/servicepackagealias"

	root := RunHTTPDSL(t, httpServicePackageAliasDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func TestHTTPServicePackageAliasExampleCLIImports(t *testing.T) {
	root := RunHTTPDSL(t, httpServicePackageAliasExampleDSL)
	services := CreateHTTPServices(root)
	files := ExampleCLIFiles("example.com/servicepackagealias/gen", services)
	require.Len(t, files, 1)

	header := codegen.HeaderDataForSection(files[0].HeaderSection())
	require.NotNil(t, header)
	aliases := make(map[string]string)
	for _, spec := range header.Imports {
		aliases[spec.Path] = spec.Name
	}
	require.Equal(t, "strconvsvc", aliases["example.com/servicepackagealias/gen/strconv"])
	require.Equal(t, "bodysvc", aliases["example.com/servicepackagealias/gen/body"])
	require.Equal(t, "bodysvcsvc", aliases["example.com/servicepackagealias/gen/bodysvc"])

	serverFiles := ExampleServerFiles("example.com/servicepackagealias/gen", services)
	require.Len(t, serverFiles, 1)
	serverHeader := codegen.HeaderDataForSection(serverFiles[0].HeaderSection())
	require.NotNil(t, serverHeader)
	localNames := make(map[string]struct{}, len(serverHeader.Imports))
	for _, spec := range serverHeader.Imports {
		name := spec.Name
		if name == "" || name == "_" || name == "." {
			continue
		}
		_, exists := localNames[name]
		require.Falsef(t, exists, "duplicate example-server import name %q", name)
		localNames[name] = struct{}{}
	}
}

func TestHTTPServicePackageAliasAggregateExamplesCompile(t *testing.T) {
	const modulePath = "example.com/servicepackagealiasexamples"
	root := RunHTTPDSL(t, httpServicePackageAliasAggregateDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	services := CreateHTTPServices(root)
	files := ExampleServerFiles(modulePath+"/gen", services)
	renderGeneratedFiles(t, dir, files)

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./cmd/api")

	cli := ExampleCLIFiles(modulePath+"/gen", services)[0]
	header := codegen.HeaderDataForSection(cli.HeaderSection())
	aliases := make(map[string]string)
	for _, spec := range header.Imports {
		aliases[spec.Path] = spec.Name
	}
	require.Equal(t, "contextsvc", aliases[modulePath+"/gen/context"])
	require.Equal(t, "json_", aliases[modulePath+"/gen/json_"])
	require.Equal(t, "clisvc", aliases[modulePath+"/gen/cli"])
}

func httpServicePackageAliasDSL() {
	names := append([]string{"ordinary"}, httpServicePackageCollisionNames...)
	for _, serviceName := range names {
		Service(serviceName, func() {
			Method("create", func() {
				Payload(func() {
					Attribute("name", String)
					Required("name")
				})
				Result(func() {
					Attribute("id", String)
					Required("id")
				})
				HTTP(func() {
					POST("/items")
				})
			})
		})
	}
}

func httpServicePackageAliasExampleDSL() {
	API("service-package-alias", func() {
		Server("api", func() {
			Services("strconv", "body", "bodysvc")
			Host("development", func() {
				URI("http://localhost:8080")
			})
		})
	})
	httpServicePackageAliasDSL()
}

func httpServicePackageAliasAggregateDSL() {
	API("service-package-alias-aggregate", func() {
		Server("api", func() {
			Services("context", "json", "cli")
			Host("development", func() {
				URI("http://localhost:8080")
			})
		})
	})
	for _, serviceName := range []string{"context", "json", "cli"} {
		Service(serviceName, func() {
			Method("show", func() {
				Result(String)
				HTTP(func() {
					GET("/items")
				})
			})
		})
	}
}
