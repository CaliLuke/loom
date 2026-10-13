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
			if output := os.Getenv("LOOM_BINARY_METADATA_OUTPUT"); output != "" {
				dir = filepath.Join(output, "security", map[bool]string{false: "basic", true: "session"}[session])
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			renderGRPCResponseContractModule(t, dir, "example.com/scope", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "endpoint_test.go"), []byte(testdata.SecurityScopeEndpointTest(session)), 0o600))
			if session {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(sessionMetadataHarness), 0o600))
			}
			if os.Getenv("LOOM_BINARY_METADATA_GENERATE_ONLY") == "1" {
				return
			}
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "build", "./...")
			runGRPCGoCommand(t, dir, "vet", "./...")
			runGRPCGoCommand(t, dir, "test", "-race", "./...", "-run", "^TestSelected(Endpoint|Metadata)$")
		})
	}
}
