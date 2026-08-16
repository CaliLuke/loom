package generator

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTestScaffold(t *testing.T) {
	root := expr.RunDSL(t, testdata.ServerNoPayloadNoResultDSL)
	files, err := TestScaffold("example.com/widgets/gen", []eval.Root{root})
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join("internal", "contracttest", "service_no_payload_no_result_http_test.go"), files[0].Path)
	require.True(t, files[0].SkipExist)
}

func TestTestScaffoldIncludesGRPCContracts(t *testing.T) {
	root := expr.RunDSL(t, grpctestdata.UnaryRPCNoPayloadDSL)
	files, err := TestScaffold("example.com/widgets/gen", []eval.Root{root})
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join("internal", "contracttest", "service_unary_rpc_no_payload_grpc_test.go"), files[0].Path)
	require.True(t, files[0].SkipExist)
}

func TestGeneratorsIncludesTestScaffold(t *testing.T) {
	generators, err := generators("test-scaffold")
	require.NoError(t, err)
	require.Len(t, generators, 1)
}
