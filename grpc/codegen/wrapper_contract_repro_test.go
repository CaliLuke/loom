package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

const wrapperContractReproHarness = `package probe

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"example.com/wrapperprobe/gen/dual"
	"example.com/wrapperprobe/gen/grpc/dual/client"
	pb "example.com/wrapperprobe/gen/grpc/dual/pb"
	"example.com/wrapperprobe/gen/grpc/dual/server"
)

func TestStrictInnerLength(t *testing.T) {
	request := client.NewProtoEchoRequest(&dual.Data{Strict: [][]int{{1}}})
	data, err := proto.Marshal(request)
	require.NoError(t, err)
	decoded := &pb.EchoRequest{}
	require.NoError(t, proto.Unmarshal(data, decoded))
	require.ErrorContains(t, server.ValidateEchoRequest(decoded), "length")
}
`

func TestWrapperContractRepro(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		data := Type("Data", func() {
			Field(1, "free", ArrayOf(ArrayOf(Int)))
			Field(2, "strict", ArrayOf(ArrayOf(Int), func() {
				MinLength(2)
			}))
		})
		Service("dual", func() {
			Method("echo", func() {
				Payload(data)
				Result(data)
				GRPC(func() {
				})
			})
		})
	})
	runWrapperContractModule(t, root, wrapperContractReproHarness)
}

func runWrapperContractModule(t *testing.T, root *expr.RootExpr, harness string) {
	t.Helper()
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/wrapperprobe", root, resolveGRPCLoomSource(t))
	path := filepath.Join(dir, "go.mod")
	mod, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427")), 0o600))
	path = filepath.Join(dir, "internal", "probe")
	require.NoError(t, os.MkdirAll(path, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(path, "probe_test.go"), []byte(harness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "./internal/probe", "-v")
}
