package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
)

func TestGRPCServerEmbeddingImportIdentity(t *testing.T) {
	root := RunGRPCDSL(t, grpcMemberDesign([]string{"UnimplementedMembersServer"}, true, false, true))
	services := CreateGRPCServices(root)
	neutral := services.ServicesData.Get("members")
	require.NoError(t, servicecodegen.SetUserTypeImports("example.com/grpcmembers/gen", neutral))
	original := services.Get("members")
	require.Equal(t, "unimplementedMembersServerEmbedded2", original.serverEmbeddingAlias)
	imports := make([]*codegen.ImportSpec, 0, 1+len(original.Service.UserTypeImports))
	imports = append(imports, &codegen.ImportSpec{Path: "example.com/grpcmembers/gen/grpc/members/pb", Name: original.PkgName})
	for _, spec := range original.Service.UserTypeImports {
		copy := *spec
		if copy.Name == "unimplementedMembersServer" {
			copy.Name = original.PkgName
		}
		imports = append(imports, &copy)
	}
	renamed, actual := services.fileData("members", imports)
	require.NotSame(t, original, renamed)
	require.Equal(t, "unimplementedMembersServer", renamed.serverEmbeddingAlias, "the old type import binding is free after qualification")
	for _, spec := range actual {
		require.NotEqual(t, renamed.serverEmbeddingAlias, spec.Name)
	}
	require.Equal(t, original.Endpoints[0].RPCGoName, renamed.Endpoints[0].RPCGoName)
	require.Equal(t, original.Endpoints[0].ServerHandler, renamed.Endpoints[0].ServerHandler)
	require.Contains(t, sectionCode(t, grpcStreamSendSection(renamed.Endpoints[0].ServerStream)), original.PkgName+"2.Item")
	require.Contains(t, sectionCode(t, grpcServerStructSection(renamed)), "type unimplementedMembersServer = "+original.PkgName+".UnimplementedMembersServer")
}
