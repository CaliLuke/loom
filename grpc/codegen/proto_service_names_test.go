package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesNonASCIIServiceNames checks that service and rpc names
// derived from non-ASCII or digit-bearing design names are valid ASCII
// protocol buffer identifiers that protoc accepts.
func TestProtoFilesNonASCIIServiceNames(t *testing.T) {
	root := RunGRPCDSL(t, testdata.NonASCIIServiceNamesDSL)
	fs := ProtoFiles("", CreateGRPCServices(root))
	require.Len(t, fs, 1)
	assert.Equal(t, filepath.Join("gen", "grpc", "cafu00e9", "pb", "loomgen__cafu00e9.proto"), fs[0].Path)
	sections := fs[0].AllSections()
	require.GreaterOrEqual(t, len(sections), 3)
	code := sectionCode(t, sections[1:]...)

	assert.Contains(t, code, "package cafu00e9;")
	assert.Contains(t, code, "service Caf {")
	assert.Contains(t, code, "rpc AAdir (AAdirRequest) returns (AAdirResponse);")
	assert.Contains(t, code, "rpc FlSs (stream FlSsStreamingRequest) returns (stream FlSsResponse);")
	assert.Contains(t, code, "rpc Get3d (Get3DRequest) returns (Get3DResponse);")
	assert.Contains(t, code, "rpc Message (MessageRequest) returns (MessageResponse);")
	for line := range strings.Lines(code) {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		for _, r := range line {
			assert.Less(t, r, rune(0x80), "non-ASCII rune %q in proto line %q", r, line)
		}
	}
	testutil.AssertString(t, "testdata/golden/proto_protofiles-non-ascii-service-names.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesDigitServiceNames checks that the protocol buffer names of
// ASCII services and rpcs whose names contain a digit followed by a lowercase
// letter are unchanged, which keeps their wire paths stable.
func TestProtoFilesDigitServiceNames(t *testing.T) {
	root := RunGRPCDSL(t, testdata.DigitServiceNamesDSL)
	fs := ProtoFiles("", CreateGRPCServices(root))
	require.Len(t, fs, 1)
	sections := fs[0].AllSections()
	require.GreaterOrEqual(t, len(sections), 3)
	code := sectionCode(t, sections[1:]...)

	assert.Contains(t, code, "package calc2go;")
	assert.Contains(t, code, "service Calc2go {")
	assert.Contains(t, code, "rpc Get3d (Get3DRequest) returns (Get3DResponse);")
	assert.Contains(t, code, "rpc V2betaList (V2BetaListRequest) returns (V2BetaListResponse);")
	assert.Contains(t, code, "rpc Sync2way (stream Sync2WayStreamingRequest) returns (stream Sync2WayResponse);")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-digit-service-names.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

func TestProtoServiceName(t *testing.T) {
	cases := []struct {
		Name     string
		GoName   string
		Expected string
	}{
		{"ascii unchanged", "Calc2go", "Calc2go"},
		{"ascii digit lowercase", "V2betaList", "V2betaList"},
		{"latin accent", "Café", "Caf"},
		{"inner accent", "Añadir", "AAdir"},
		{"no ascii letter", "Val日本", "Val"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			assert.Equal(t, c.Expected, protoServiceName(c.GoName))
		})
	}
}

// TestDigitServiceNamesGeneratedModuleServes checks that the generated Go
// server and client of an ASCII service with digit-bearing names use the Go
// names protoc-gen-go derives from the unchanged protocol buffer names.
func TestDigitServiceNamesGeneratedModuleServes(t *testing.T) {
	root := RunGRPCDSL(t, testdata.DigitServiceNamesDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/digits", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service_names_test.go"), []byte(digitServiceNamesHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", ".")
}

// TestNonASCIIServiceNamesGeneratedModuleServes checks that the generated Go
// server and client use the names protoc-gen-go generates for the service
// and its rpcs: every rpc reaches the service implementation instead of the
// embedded Unimplemented server.
func TestNonASCIIServiceNamesGeneratedModuleServes(t *testing.T) {
	root := RunGRPCDSL(t, testdata.NonASCIIServiceNamesDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/nonascii", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service_names_test.go"), []byte(nonASCIIServiceNamesHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", ".")
}

const nonASCIIServiceNamesHarness = `package nonascii

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"example.com/nonascii/gen/cafu00e9"
	cafclient "example.com/nonascii/gen/grpc/cafu00e9/client"
	cafpb "example.com/nonascii/gen/grpc/cafu00e9/pb"
	cafserver "example.com/nonascii/gen/grpc/cafu00e9/server"
)

type service struct{}

func (service) Añadir(_ context.Context, p *cafu00e9.AñadirPayload) (*cafu00e9.AñadirResult, error) {
	total := len(*p.Nombre)
	return &cafu00e9.AñadirResult{Total: &total}, nil
}

func (service) Flüss(_ context.Context, stream cafu00e9.FlüssServerStream) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&cafu00e9.FlüssResult{Echo: in.Wert}); err != nil {
			return err
		}
	}
}

func (service) Get3d(_ context.Context, p string) (string, error) {
	return "3d:" + p, nil
}

func (service) Message(_ context.Context, p string) (string, error) {
	return "message:" + p, nil
}

func TestServiceNames(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	cafpb.RegisterCafServer(server, cafserver.New(cafu00e9.NewEndpoints(service{}), nil, nil))
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	ctx := context.Background()
	client := cafclient.NewClient(conn)

	nombre := "loom"
	res, err := client.Añadir()(ctx, &cafu00e9.AñadirPayload{Nombre: &nombre})
	require.NoError(t, err)
	require.Equal(t, 4, *res.(*cafu00e9.AñadirResult).Total)

	res, err = client.Get3d()(ctx, "x")
	require.NoError(t, err)
	require.Equal(t, "3d:x", res)

	res, err = client.Message()(ctx, "y")
	require.NoError(t, err)
	require.Equal(t, "message:y", res)

	res, err = client.Flüss()(ctx, nil)
	require.NoError(t, err)
	stream := res.(cafu00e9.FlüssClientStream)
	wert := "w"
	require.NoError(t, stream.Send(&cafu00e9.FlüssStreamingPayload{Wert: &wert}))
	echo, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, "w", *echo.Echo)
	require.NoError(t, stream.Close())
}
`

const digitServiceNamesHarness = `package digits

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"example.com/digits/gen/calc2go"
	calc2goclient "example.com/digits/gen/grpc/calc2go/client"
	calc2gopb "example.com/digits/gen/grpc/calc2go/pb"
	calc2goserver "example.com/digits/gen/grpc/calc2go/server"
)

type service struct{}

func (service) Get3d(_ context.Context, p string) (string, error) {
	return "3d:" + p, nil
}

func (service) V2betaList(_ context.Context, p string) (string, error) {
	return "v2:" + p, nil
}

func (service) Sync2way(_ context.Context, stream calc2go.Sync2wayServerStream) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.Send("echo:" + in); err != nil {
			return err
		}
	}
}

func TestServiceNames(t *testing.T) {
	require.Equal(t, "/calc2go.Calc2go/Get3d", calc2gopb.Calc2Go_Get3D_FullMethodName)
	require.Equal(t, "/calc2go.Calc2go/V2betaList", calc2gopb.Calc2Go_V2BetaList_FullMethodName)
	require.Equal(t, "/calc2go.Calc2go/Sync2way", calc2gopb.Calc2Go_Sync2Way_FullMethodName)

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	calc2gopb.RegisterCalc2GoServer(server, calc2goserver.New(calc2go.NewEndpoints(service{}), nil, nil))
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	ctx := context.Background()
	client := calc2goclient.NewClient(conn)

	res, err := client.Get3d()(ctx, "x")
	require.NoError(t, err)
	require.Equal(t, "3d:x", res)

	res, err = client.V2betaList()(ctx, "y")
	require.NoError(t, err)
	require.Equal(t, "v2:y", res)

	res, err = client.Sync2way()(ctx, nil)
	require.NoError(t, err)
	stream := res.(calc2go.Sync2wayClientStream)
	require.NoError(t, stream.Send("z"))
	echo, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, "echo:z", echo)
	require.NoError(t, stream.Close())
}
`

// TestReservedNameRPCGoNames checks that the Go code calls and implements an
// rpc named after a protocol buffer keyword with the name protoc-gen-go
// generates, without the underscore protoBufify adds to field names.
func TestReservedNameRPCGoNames(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MethodWithReservedNameDSL)
	services := CreateGRPCServices(root)
	var code strings.Builder
	for _, f := range append(ClientFiles("", services), ServerFiles("", services)...) {
		for _, s := range f.AllSections() {
			code.WriteString(sectionCode(t, s))
		}
	}
	assert.Contains(t, code.String(), "grpccli.String(")
	assert.Contains(t, code.String(), "func (s *Server) String(")
	assert.NotContains(t, code.String(), "String_(")
}

// TestProtoNameCollisions checks that generation fails with an error naming
// both design methods when they map to the same protocol buffer rpc name, or
// to the same generated message name with different fields, and that
// methods whose generated messages have the same name and fields share them.
func TestProtoNameCollisions(t *testing.T) {
	unary := func(name string, payload, result any) func() {
		return func() {
			Method(name, func() {
				Payload(payload)
				Result(result)
				GRPC(func() {})
			})
		}
	}
	nested := func(name string, fields func()) func() {
		return func() {
			Method(name, func() {
				Payload(func() {
					Field(1, "x", fields)
				})
				Result(String)
				GRPC(func() {})
			})
		}
	}
	cases := []struct {
		Name     string
		Methods  []func()
		Expected string
	}{
		{"non-ASCII separator", []func(){unary("añadir", String, String), unary("a_adir", String, String)},
			`methods "añadir" and "a_adir" of service "Café" both map to protocol buffer rpc "AAdir"`},
		{"no ASCII letter", []func(){unary("日本", String, String), unary("中国", String, String)},
			`methods "日本" and "中国" of service "Café" both map to protocol buffer rpc "Val"`},
		{"same message fields", []func(){unary("get3d", String, String), unary("get3_d", String, String)}, ""},
		{"different request fields", []func(){unary("get3d", String, String), unary("get3_d", Int, String)},
			`methods "get3d" and "get3_d" of service "Café" both map to protocol buffer message "Get3DRequest" with different fields`},
		{"different nested field number", []func(){
			nested("get3d", func() { Field(1, "a", String) }),
			nested("get3_d", func() { Field(2, "a", String) }),
		}, `methods "get3d" and "get3_d" of service "Café" both map to protocol buffer message "Get3DRequest" with different fields`},
		{"different nested requiredness", []func(){
			nested("get3d", func() { Field(1, "a", String) }),
			nested("get3_d", func() { Field(1, "a", String); Required("a") }),
		}, `methods "get3d" and "get3_d" of service "Café" both map to protocol buffer message "Get3DRequest" with different fields`},
		{"same nested fields", []func(){
			nested("get3d", func() { Field(1, "a", String) }),
			nested("get3_d", func() { Field(1, "a", String) }),
		}, ""},
		{"streaming request", []func(){
			func() {
				Method("upload", func() {
					StreamingPayload(String)
					Result(String)
					GRPC(func() {})
				})
			},
			unary("upload_streaming", Int, String),
		}, `methods "upload" and "upload_streaming" of service "Café" both map to protocol buffer message "UploadStreamingRequest" with different fields`},
		{"error", []func(){
			func() {
				Method("a", func() {
					Payload(String)
					Result(String)
					Error("b_c", func() {
						Field(1, "x", String)
					})
					GRPC(func() {
						Response("b_c", CodeInternal)
					})
				})
			},
			func() {
				Method("a_b", func() {
					Payload(String)
					Result(String)
					Error("c", func() {
						Field(1, "y", Int)
					})
					GRPC(func() {
						Response("c", CodeInternal)
					})
				})
			},
		}, `methods "a" and "a_b" of service "Café" both map to protocol buffer message "ABCError" with different fields`},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, func() {
				Service("Café", func() {
					for _, m := range c.Methods {
						m()
					}
				})
			})
			err := generationError(func() { ProtoFiles("", CreateGRPCServices(root)) })
			if c.Expected == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.Expected)
		})
	}
}

// generationError returns the error that fn panics with, if any.
func generationError(fn func()) (err error) {
	defer func() {
		err = codegen.RecoverPanic(recover())
	}()
	fn()
	return nil
}
