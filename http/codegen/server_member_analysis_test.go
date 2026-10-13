package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/dsl"
)

func TestServerMembersRetainFileImportIdentity(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	for _, expressions := range []*ServicesData{
		CreateHTTPServices(root),
		NewServicesData(servicecodegen.NewServicesData(root), &root.API.JSONRPC.HTTPExpr),
	} {
		for _, service := range expressions.Expressions.Services {
			require.NoError(t, servicecodegen.SetUserTypeImports("example.com/imports/gen", expressions.ServicesData.Get(service.Name())))
			original := expressions.Get(service.Name())
			imports := append([]*codegen.ImportSpec{{Path: "example.com/conflict", Name: "common"}}, original.Service.UserTypeImports...)
			renamed, imports := expressions.FileData(service.Name(), imports)
			require.NotSame(t, original, renamed, service.Name())
			require.NotEqual(t, original.Service.UserTypeImports, imports)
			for i, endpoint := range original.Endpoints {
				require.Equal(t, endpoint.ServerHandler, renamed.Endpoints[i].ServerHandler)
				require.Equal(t, endpoint.ServerEndpoint, renamed.Endpoints[i].ServerEndpoint)
				require.Equal(t, endpoint.Method.VarName, renamed.Endpoints[i].Method.VarName)
			}
		}
	}
}

func TestServerMembersRespectConcreteReceiver(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		dsl.Service("plain", func() {
			for _, name := range []string{"handler", "methods", "streamHandler", "read"} {
				dsl.Method(name, func() {
					dsl.HTTP(func() {
						dsl.GET("/" + name)
					})
				})
			}
		})
		dsl.Service("files", func() {
			dsl.Files("/one", "Mounts")
			dsl.Files("/two", "MountMounts")
		})
	})
	services := CreateHTTPServices(root)
	for _, endpoint := range services.Get("plain").Endpoints {
		require.Equal(t, endpoint.Method.VarName, endpoint.ServerHandler, "JSON-RPC-only members must not occupy the HTTP receiver")
		require.Empty(t, endpoint.ServerEndpoint)
	}
	files := services.Get("files")
	require.Equal(t, "Mounts2", files.FileServers[0].VarName)
	require.Equal(t, "MountMounts", files.FileServers[1].VarName, "package mount helpers are not receiver members")
}
