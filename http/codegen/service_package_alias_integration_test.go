package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

var httpServicePackageCollisionNames = []string{
	"body",
	"bytes",
	"c",
	"ctx",
	"err",
	"io",
	"loom",
	"loomhttp",
	"v",
}

func TestHTTPServicePackageAliasAnalysis(t *testing.T) {
	root := RunHTTPDSL(t, httpServicePackageAliasDSL)
	services := CreateHTTPServices(root)

	require.Equal(t, "ordinary", services.ServicesData.Get("ordinary").PkgName)
	require.Equal(t, "ordinary", services.Get("ordinary").Service.PkgName)
	for _, name := range httpServicePackageCollisionNames {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, name, services.ServicesData.Get(name).PkgName)
			require.Equal(t, name+"svc", services.Get(name).Service.PkgName)
			require.Equal(t, name+"svc", services.Get(name).Endpoints[0].ServicePkgName)
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
