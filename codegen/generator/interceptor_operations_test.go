package generator

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestGeneratedInterceptorServerOperations(t *testing.T) {
	for _, mode := range []string{"rpc-sse", "rpc-bidi", "rpc-request-reply"} {
		t.Run(mode, func(t *testing.T) {
			harness, err := os.ReadFile("testdata/interceptor_operations_" + strings.ReplaceAll(mode, "-", "_") + ".go.txt")
			require.NoError(t, err)
			runDesignHarness(t, "example.com/interceptorcontracts", testdata.InterceptorContractStreamDSL(mode), string(harness))
		})
	}
}
