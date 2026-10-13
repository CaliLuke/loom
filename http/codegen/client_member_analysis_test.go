package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
)

func TestClientMembersRetainFileImportIdentity(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.UserTypePackageTransportsDSL)
	for _, services := range []*ServicesData{
		CreateHTTPServices(root),
		NewServicesData(servicecodegen.NewServicesData(root), &root.API.JSONRPC.HTTPExpr),
	} {
		for _, service := range services.Expressions.Services {
			require.NoError(t, servicecodegen.SetUserTypeImports("example.com/imports/gen", services.ServicesData.Get(service.Name())))
			original := services.Get(service.Name())
			imports := append([]*codegen.ImportSpec{{Path: "example.com/conflict", Name: "common"}}, original.Service.UserTypeImports...)
			renamed, imports := services.FileData(service.Name(), imports)
			require.NotSame(t, original, renamed)
			require.NotEqual(t, original.Service.UserTypeImports, imports)
			for i, endpoint := range original.Endpoints {
				got := renamed.Endpoints[i]
				require.Equal(t, endpoint.EndpointInit, got.EndpointInit)
				require.Equal(t, endpoint.StreamEndpointInit, got.StreamEndpointInit)
				require.Equal(t, endpoint.ClientDoer, got.ClientDoer)
				require.Equal(t, endpoint.RequestInit.Name, got.RequestInit.Name)
				require.Equal(t, endpoint.Method.VarName, got.Method.VarName)
			}
			require.Len(t, renamed.clientGroups, len(original.clientGroups))
			for i, group := range original.clientGroups {
				require.Equal(t, group.FieldName, renamed.clientGroups[i].FieldName)
				require.Equal(t, group.Name, renamed.clientGroups[i].Name)
			}
		}
	}
}
