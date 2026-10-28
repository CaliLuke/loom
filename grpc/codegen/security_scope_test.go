package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
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

func TestGRPCSecurityScopeMatrixProjection(t *testing.T) {
	root := RunGRPCDSL(t, testdata.SecurityScopeMatrixDSL("grpc"))
	services := CreateGRPCServices(root)
	apiEndpoint := services.Get("api_scope").Endpoint("read")
	require.Equal(t, []string{"basic"}, grpcRequirementSchemeNames(apiEndpoint.Method.Requirements))

	for _, tc := range []struct {
		method string
		want   []string
	}{
		{method: "inherited", want: []string{"jwt"}},
		{method: "override", want: []string{"key"}},
		{method: "alternatives", want: []string{"basic", "jwt"}},
		{method: "open"},
	} {
		endpoint := services.Get("scope").Endpoint(tc.method)
		require.NotNil(t, endpoint)
		require.Equal(t, tc.want, grpcRequirementSchemeNames(endpoint.Method.Requirements), tc.method)
	}
}

func TestGRPCSecurityScopeMatrixRuntime(t *testing.T) {
	root := RunGRPCDSL(t, testdata.SecurityScopeMatrixDSL("grpc"))
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/scopematrix", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_matrix_test.go"), []byte(grpcSecurityScopeMatrixHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-race", "./...", "-run", "^TestSecurityScopeMatrix$")
}

func grpcRequirementSchemeNames(requirements service.RequirementsData) []string {
	var names []string
	for _, requirement := range requirements {
		for _, scheme := range requirement.Schemes {
			names = append(names, scheme.SchemeName)
		}
	}
	return names
}

func TestGRPCPerSchemeSecurityProjection(t *testing.T) {
	root := RunGRPCDSL(t, testdata.SecurityPerSchemeDSL("grpc"))
	endpoint := CreateGRPCServices(root).Get("per_scheme").Endpoint("read")
	requirements := endpoint.Method.Requirements
	require.Len(t, requirements, 3)
	require.Equal(t, []string{"read"}, requirements[0].Schemes[0].RequiredScopes)
	require.Equal(t, []string{"write"}, requirements[0].Schemes[1].RequiredScopes)
	require.Equal(t, []string{"read"}, requirements[1].Schemes[0].RequiredScopes)
	require.Empty(t, requirements[2].Schemes)
}
