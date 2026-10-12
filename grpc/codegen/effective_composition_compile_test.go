package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestEffectiveCompositionGeneratedGRPC(t *testing.T) {
	root := RunGRPCDSL(t, httpdata.EffectiveCompositionDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/composition", root)
	generated := readGeneratedGRPCGo(t, dir)
	require.Contains(t, generated, "100")
	require.Contains(t, generated, "InvalidRangeError")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composition_test.go"), []byte(effectiveCompositionGRPCHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", ".", "./gen/...")
}

const effectiveCompositionGRPCHarness = `package composition_test

import (
 "testing"
 "github.com/stretchr/testify/require"
 pb "example.com/composition/gen/grpc/composition/pb"
 server "example.com/composition/gen/grpc/composition/server"
)

func TestEffectiveComposition(t *testing.T) {
 label := "ok"
 for _, test := range []struct { value int64; rejected bool }{
  {-1, true}, {0, false}, {50, false}, {100, false}, {101, true},
 } {
  request := &pb.ReadRequest{Count: &test.value, Label: &label}
  err := server.ValidateReadRequest(request)
  if test.rejected {
   require.Error(t, err)
  } else {
   require.NoError(t, err)
  }
 }
 require.Error(t, server.ValidateReadRequest(&pb.ReadRequest{Label: &label}))
 count := int64(50)
 require.Equal(t, 5, server.NewReadPayload(&pb.ReadRequest{Count: &count, Label: &label}).Retries)
}
`
