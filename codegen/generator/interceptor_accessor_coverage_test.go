package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestGeneratedInterceptorAccessorCoverage(t *testing.T) {
	for _, kind := range []string{"unary", "streaming"} {
		harness, err := os.ReadFile("testdata/interceptor_accessor_" + kind + ".go.txt")
		require.NoError(t, err)
		for _, mode := range []string{"single", "multiple"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				server := []string{"a"}
				if mode == "multiple" {
					server = append(server, "b")
				}
				design := testdata.InterceptorAccessorCoverageDSL(server, []string{"a", "c"}, kind == "streaming")
				runDesignHarness(t, "example.com/coverage", design, string(harness))
			})
		}
	}
}

func TestGeneratedInterceptorAccessorAliases(t *testing.T) {
	harness, err := os.ReadFile("testdata/interceptor_accessor_aliases.go.txt")
	require.NoError(t, err)
	runDesignHarness(t, "example.com/coverage", testdata.InterceptorAccessorAliasDSL, string(harness))
}
