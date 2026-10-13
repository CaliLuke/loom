package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestGeneratedSecurityRequiredScopes(t *testing.T) {
	harness, err := os.ReadFile("testdata/security_required_scopes.go.txt")
	require.NoError(t, err)
	runDesignHarness(t, "example.com/scopetest", testdata.SecurityRequiredScopesDSL, string(harness))
}
