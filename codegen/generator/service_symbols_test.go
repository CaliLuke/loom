package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestServiceReservedSymbolsCompile(t *testing.T) {
	dir := t.TempDir()
	renderDesignModule(t, dir, "example.com/symbols", testdata.ServiceReservedSymbolsDSL)
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

func TestConditionalServiceSymbolsCompile(t *testing.T) {
	for _, mode := range []string{"plain", "secure", "sse", "websocket", "interceptors"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			renderDesignModule(t, dir, "example.com/symbols", func() {
				testdata.ServiceConditionalSymbolsDSL(mode)
			})
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

func TestExternalServiceSymbolsCompile(t *testing.T) {
	dir := t.TempDir()
	renderDesignModule(t, dir, "example.com/symbols", testdata.ServiceExternalSymbolsDSL)
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

func TestServiceMethodNamespaceControlsCompile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"fixed spellings", testdata.ServiceFixedSpellingMethodsDSL},
		{"actual member collisions", testdata.ServiceMemberFieldsDSL},
		{"type and derived helper spellings", testdata.ServiceMethodTypeCouplingDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			renderDesignModule(t, dir, "example.com/symbols", tc.design)
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

func TestServiceSymbolGRPCDescriptors(t *testing.T) {
	dir := t.TempDir()
	renderDesignModule(t, dir, "example.com/symbols", testdata.ServiceMethodTypeCouplingDSL)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "symbols_test.go"), []byte(serviceSymbolGRPCDescriptorTest), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const serviceSymbolGRPCDescriptorTest = `package symbols_test

import (
    "context"
    "errors"
    "testing"

    pb "example.com/symbols/gen/grpc/symbols/pb"
    "google.golang.org/grpc"
)

type recordingConn struct {
    method string
}

func (*recordingConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
    return errors.New("unexpected unary call")
}

func (c *recordingConn) NewStream(_ context.Context, _ *grpc.StreamDesc, method string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
    c.method = method
    return nil, errors.New("recorded stream call")
}

func TestAuthoredMethodIdentity(t *testing.T) {
    service := pb.File_loomgen_symbols_symbols_proto.Services().ByName("Symbols")
    if service.Methods().ByName("Foo") == nil || service.Methods().ByName("FooEndpoint") != nil {
        t.Fatal("protobuf descriptors must use the independent Foo method identity")
    }
    if pb.Symbols_ServiceDesc.Streams[0].StreamName != "Foo" {
        t.Fatal("server stream descriptor does not match Foo")
    }
    conn := &recordingConn{}
    _, err := pb.NewSymbolsClient(conn).Foo(context.Background(), &pb.FooRequest{})
    if pb.Symbols_Foo_FullMethodName != "/symbols.Symbols/Foo" {
        t.Fatal("client full method constant does not match Foo")
    }
    if err == nil || conn.method != pb.Symbols_Foo_FullMethodName {
        t.Fatalf("client method=%q error=%v", conn.method, err)
    }
}
`
