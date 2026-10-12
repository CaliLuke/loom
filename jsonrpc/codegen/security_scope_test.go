package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr/testdata"
)

func TestJSONRPCSecurityScope(t *testing.T) {
	for _, session := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "session"}[session], func(t *testing.T) {
			root := RunJSONRPCDSL(t, testdata.SecurityScopeDSL("jsonrpc", session))
			endpoint := CreateJSONRPCServices(root).Get("scope").Endpoint("read")
			if session {
				require.Nil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Len(t, endpoint.Method.Requirements, 2)
			} else {
				require.Len(t, endpoint.Method.Requirements, 1)
				require.NotNil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Empty(t, endpoint.Payload.Request.Cookies)
				require.NotContains(t, endpoint.Method.PayloadDef, "Auth *string")
			}
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/scope", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "endpoint_test.go"), []byte(testdata.SecurityScopeEndpointTest(session)), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "./...")
		})
	}
}
