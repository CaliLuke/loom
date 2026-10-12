package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr/testdata"
)

func TestGRPCSecurityScope(t *testing.T) {
	for _, session := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "session"}[session], func(t *testing.T) {
			root := RunGRPCDSL(t, testdata.SecurityScopeDSL("grpc", session))
			endpoint := CreateGRPCServices(root).Get("scope").Endpoints[0]
			if session {
				require.Nil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Len(t, endpoint.Method.Requirements, 2)
			} else {
				require.Len(t, endpoint.Method.Requirements, 1)
				require.NotNil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Nil(t, grpcScheme(endpoint.MetadataSchemes, "jwt"))
				require.Nil(t, grpcScheme(endpoint.MetadataSchemes, "key"))
				require.NotContains(t, endpoint.Method.PayloadDef, "Auth *string")
			}
			dir := t.TempDir()
			renderGRPCResponseContractModule(t, dir, "example.com/scope", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "endpoint_test.go"), []byte(testdata.SecurityScopeEndpointTest(session)), 0o600))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "test", "./...", "-run", "^TestSelectedEndpoint$")
		})
	}
}
