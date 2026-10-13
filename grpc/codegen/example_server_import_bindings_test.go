package codegen

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestGRPCExampleServerRetainsAllImportRoles(t *testing.T) {
	data := &ServiceData{
		Service: &service.Data{
			VarName: "ordinary",
			PkgName: "original",
			Methods: []*service.MethodData{{Name: "echo"}},
		},
		PkgName:         "originalpb",
		ServerInterface: "OrdinaryServer",
		Endpoints:       []*EndpointData{{Method: &service.MethodData{Name: "echo"}}},
	}
	section := grpcExampleServerSection([]exampleServerServiceData{{
		data:           data,
		serviceImport:  "service_alias",
		serverImport:   "server_alias",
		protobufImport: "protobuf_alias",
	}})
	code := codegen.SectionCode(t, section)
	for _, want := range []string{
		"ordinaryEndpoints *service_alias.Endpoints",
		"ordinaryServer *server_alias.Server",
		"ordinaryServer = server_alias.New(ordinaryEndpoints, nil)",
		"protobuf_alias.RegisterOrdinaryServer(srv, ordinaryServer)",
	} {
		require.Contains(t, code, want)
	}
	require.NotContains(t, code, "original.")
	require.NotContains(t, code, "originalsvr.")
	require.NotContains(t, code, "originalpb.")
	require.Equal(t, "original", data.Service.PkgName)
	require.Equal(t, "originalpb", data.PkgName)
}

func TestGRPCExampleServerHostBindingsAreIsolated(t *testing.T) {
	root := RunGRPCDSL(t, testdata.GRPCLocalSymbolsHosts)
	services := CreateGRPCServices(root)
	before := make(map[string][2]string)
	for _, name := range []string{"p", "psvcsvr", "req"} {
		data := services.Get(name)
		before[name] = [2]string{data.Service.PkgName, data.PkgName}
	}
	render := func() map[string]string {
		files := ExampleServerFiles("example.com/hosts/gen", services)
		require.Len(t, files, 2)
		result := make(map[string]string)
		for _, file := range files {
			result[file.Path] = codegen.SectionCode(t, file.Section("server-grpc-main")[0])
		}
		return result
	}
	first := render()
	require.Contains(t, first["cmd/first/grpc.go"], "psvcsvrEndpoints *psvcsvr2.Endpoints")
	require.NotContains(t, first["cmd/first/grpc.go"], "reqEndpoints")
	require.Contains(t, first["cmd/second/grpc.go"], "reqEndpoints *req.Endpoints")
	require.Contains(t, first["cmd/second/grpc.go"], "reqpb2.RegisterReqServer")
	require.NotContains(t, first["cmd/second/grpc.go"], "pEndpoints")
	// Reverse file visitation while keeping authored service declaration order.
	slices.Reverse(root.API.Servers)
	require.Equal(t, first, render())
	for name, aliases := range before {
		data := services.Get(name)
		require.Equal(t, aliases, [2]string{data.Service.PkgName, data.PkgName})
	}
}
