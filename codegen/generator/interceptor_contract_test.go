package generator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestGeneratedInterceptorMethodBindings(t *testing.T) {
	harness, err := os.ReadFile("testdata/interceptor_contract.go.txt")
	require.NoError(t, err)
	runDesignHarness(t, "example.com/interceptorcontracts", testdata.InterceptorContractStorageDSL("required"), string(harness))
}

func TestGeneratedInterceptorUnionStorage(t *testing.T) {
	harness, err := os.ReadFile("testdata/interceptor_contract_union.go.txt")
	require.NoError(t, err)
	runDesignHarness(t, "example.com/interceptorcontracts", testdata.InterceptorContractStorageDSL("union"), string(harness))
}
