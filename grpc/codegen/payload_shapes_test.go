package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestProtoFilesRPCNamedMessages checks that the service definition
// qualifies with the package the messages that have the name of an rpc, as
// protoc resolves the names of the rpc messages in the scope of the service
// first, and leaves the other messages unqualified.
func TestProtoFilesRPCNamedMessages(t *testing.T) {
	code := protoFileCode(t, testdata.PayloadShapesDSL)

	for _, want := range []string{
		"rpc Tags (.shapes.Tags) returns (.shapes.Tags);",
		"rpc Index (.shapes.Index) returns (.shapes.Index);",
		"rpc ID (.shapes.ID) returns (.shapes.ID);",
		"rpc Describe (DescribeRequest) returns (DescribeResponse);",
		"rpc Empty (EmptyRequest) returns (EmptyResponse);",
		"rpc Watch (WatchRequest) returns (stream WatchResponse);",
		"rpc Upload (stream UploadStreamingRequest) returns (UploadResponse);",
		"message Tags {\n\trepeated string field = 1;\n}",
		"message Index {\n\tmap<string, Leaf> field = 1;\n}",
		"message ID {\n\tstring field = 1;\n}",
		"message EmptyRequest {\n}",
		"message FaultProto {\n\toptional string msg = 1;\n\tstring name = 2;\n}",
	} {
		assert.Contains(t, code, want)
	}
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestProtoFilesRPCNamedStreamingMessages checks that the streaming messages
// that have the name of an rpc are qualified with the package.
func TestProtoFilesRPCNamedStreamingMessages(t *testing.T) {
	code := protoFileCode(t, func() {
		tags := Type("Tags", ArrayOf(String))
		Service("streams", func() {
			Method("tags", func() {
				Payload(tags)
				StreamingResult(tags)
				GRPC(func() {})
			})
		})
	})

	assert.Contains(t, code, "rpc Tags (.streams.Tags) returns (stream .streams.Tags);")
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestEmptyObjectConversions checks that an empty object payload or result
// is converted like any other object: the server builds the payload and the
// client the result, the stream conversions take the stream item, and the
// encoders and decoders that do not read the value they assert discard it.
func TestEmptyObjectConversions(t *testing.T) {
	root := RunGRPCDSL(t, testdata.PayloadShapesDSL)
	services := CreateGRPCServices(root)
	sd := services.Get("shapes")
	require.NotNil(t, sd)

	empty := sd.Endpoint("empty")
	require.NotNil(t, empty)
	require.NotNil(t, empty.Request.ServerConvert)
	assert.Equal(t, "NewEmptyPayload", empty.Request.ServerConvert.Init.Name)
	assert.Empty(t, empty.Request.ServerConvert.Init.Args)
	require.NotNil(t, empty.Response.ClientConvert)
	assert.Equal(t, "NewEmptyResult", empty.Response.ClientConvert.Init.Name)
	assert.Empty(t, empty.Response.ClientConvert.Init.Args)

	for _, stream := range []*StreamData{sd.Endpoint("watch").ServerStream, sd.Endpoint("watch").ClientStream} {
		require.NotNil(t, stream)
		convert := stream.SendConvert
		if convert == nil {
			convert = stream.RecvConvert
		}
		require.NotNil(t, convert, stream.Type)
		assert.Len(t, convert.Init.Args, 1, stream.Type)
	}
	for _, stream := range []*StreamData{sd.Endpoint("upload").ServerStream, sd.Endpoint("upload").ClientStream} {
		require.NotNil(t, stream)
		require.NotNil(t, stream.SendConvert, stream.Type)
		require.NotNil(t, stream.RecvConvert, stream.Type)
		assert.Len(t, stream.SendConvert.Init.Args, 1, stream.Type)
		assert.Len(t, stream.RecvConvert.Init.Args, 1, stream.Type)
	}

	client := sectionCode(t, ClientFiles("", services)[1].AllSections()[1:]...)
	assert.Contains(t, client, "\t_, ok := v.(*shapes.Nothing)\n\tif !ok {\n\t\treturn nil, loomgrpc.ErrInvalidType(\"shapes\", \"empty\", \"*shapes.Nothing\", v)\n\t}\n\treturn NewProtoEmptyRequest(), nil\n")
	assert.Contains(t, client, "\t_, ok := v.(*shapespb.EmptyResponse)\n")
	assert.Contains(t, client, "\tres := NewEmptyResult()\n")
	server := sectionCode(t, ServerFiles("", services)[1].AllSections()[1:]...)
	assert.Contains(t, server, "\t_, ok := v.(*shapes.Nothing)\n\tif !ok {\n\t\treturn nil, loomgrpc.ErrInvalidType(\"shapes\", \"empty\", \"*shapes.Nothing\", v)\n\t}\n\tresp := NewProtoEmptyResponse()\n")
	assert.Contains(t, server, "\t\tpayload = NewEmptyPayload()\n")
}

// TestClientErrorsSharedMessage checks that the client decodes the errors of
// a method that share a protocol buffer message with one case, because it
// tells errors apart by the type of the message.
func TestClientErrorsSharedMessage(t *testing.T) {
	root := RunGRPCDSL(t, testdata.PayloadShapesDSL)
	services := CreateGRPCServices(root)
	code := sectionCode(t, ClientFiles("", services)[0].AllSections()[1:]...)

	assert.Equal(t, 3, strings.Count(code, "case *shapespb.FaultProto:"), "one case in each of the fail and fail_stream endpoints and in the fail_stream Recv")
	assert.Contains(t, code, "case *shapespb.FaultProto:\n\t\t\t\treturn nil, NewFailMissingError(message)\n\t\t\tcase *loompb.ErrorResponse:")
	assert.Contains(t, code, "case *shapespb.FaultProto:\n\t\t\treturn res, NewFailStreamMissingError(message)\n\t\tcase *loompb.ErrorResponse:")
}

// TestSharedErrorMessageDifferentTypes checks that generation fails when two
// errors of a method map different types to one protocol buffer message,
// which the client could not tell apart, and succeeds when the types belong
// to different methods.
func TestSharedErrorMessageDifferentTypes(t *testing.T) {
	cases := []struct {
		name    string
		oneRPC  bool
		wantErr string
	}{
		{"same method", true, `errors "missing" and "invalid" of method "fail" of service "faults" map different types to protocol buffer message "Shared", so the client cannot tell them apart`},
		{"different methods", false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := generationError(func() {
				protoFileCode(t, func() {
					missing := Type("Missing", func() {
						Field(1, "msg", String)
						Meta("struct:name:proto", "Shared")
					})
					invalid := Type("Invalid", func() {
						Field(1, "msg", String)
						Meta("struct:name:proto", "Shared")
					})
					Service("faults", func() {
						Method("fail", func() {
							Error("missing", missing)
							if c.oneRPC {
								Error("invalid", invalid)
							}
							GRPC(func() {
								Response("missing", CodeNotFound)
								if c.oneRPC {
									Response("invalid", CodeInvalidArgument)
								}
							})
						})
						if !c.oneRPC {
							Method("fail_other", func() {
								Error("invalid", invalid)
								GRPC(func() {
									Response("invalid", CodeInvalidArgument)
								})
							})
						}
					})
				})
			})
			if c.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.wantErr)
		})
	}
}

// TestGeneratedPayloadShapes compiles the module and the client CLI
// generated for methods named after the named array, named map and primitive
// alias that they use, a result type used only by gRPC, empty object payloads
// and results, and errors that share a message, and calls every method
// through a gRPC connection.
func TestGeneratedPayloadShapes(t *testing.T) {
	const modulePath = "example.com/grpcpayloadshapes"
	root := RunGRPCDSL(t, testdata.PayloadShapesDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	for _, file := range ClientCLIFiles(modulePath+"/gen", CreateGRPCServices(root)) {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(fmt.Sprintf(payloadShapesHarness, modulePath)), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

const payloadShapesHarness = `package roundtrip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"%[1]s/gen/grpc/shapes/client"
	pb "%[1]s/gen/grpc/shapes/pb"
	"%[1]s/gen/grpc/shapes/server"
	"%[1]s/gen/shapes"
)

type service struct{}

func (service) Tags(_ context.Context, p shapes.Tags2) (shapes.Tags2, error) {
	return p, nil
}

func (service) Index(_ context.Context, p shapes.Index2) (shapes.Index2, error) {
	return p, nil
}

func (service) ID(_ context.Context, p shapes.ID2) (shapes.ID2, error) {
	return p, nil
}

func (service) Describe(_ context.Context, p string) (*shapes.Detail, error) {
	return &shapes.Detail{Name: &p}, nil
}

func (service) Empty(_ context.Context, p *shapes.Nothing) (*shapes.Nothing, error) {
	if p == nil {
		return nil, errors.New("nil payload")
	}
	return &shapes.Nothing{}, nil
}

func (service) Watch(_ context.Context, p *shapes.Nothing, stream shapes.WatchServerStream) error {
	if p == nil {
		return errors.New("nil payload")
	}
	if err := stream.Send(&shapes.Nothing{}); err != nil {
		return err
	}
	return stream.Close()
}

func (service) Upload(_ context.Context, stream shapes.UploadServerStream) error {
	count := 0
	for {
		item, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if item == nil {
			return errors.New("nil item")
		}
		count++
	}
	if count != 2 {
		return fmt.Errorf("received %%d items", count)
	}
	return stream.SendAndClose(&shapes.Nothing{})
}

func (service) Fail(_ context.Context, name string) (string, error) {
	msg := "failed " + name
	return "", &shapes.Fault{Name: name, Msg: &msg}
}

func (service) FailStream(_ context.Context, name string, _ shapes.FailStreamServerStream) error {
	msg := "failed " + name
	return &shapes.Fault{Name: name, Msg: &msg}
}

func newClient(t *testing.T) *client.Client {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterShapesServer(srv, server.New(shapes.NewEndpoints(service{}), nil, nil))
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return client.NewClient(conn)
}

func TestNamedMessages(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	name := "leaf"
	for _, call := range []struct {
		endpoint func() func(context.Context, any) (any, error)
		value    any
	}{
		{func() func(context.Context, any) (any, error) { return c.Tags() }, shapes.Tags2{"a", "b"}},
		{func() func(context.Context, any) (any, error) { return c.Index() }, shapes.Index2{"k": {Name: &name}, "empty": {}}},
		{func() func(context.Context, any) (any, error) { return c.ID() }, shapes.ID2("id")},
	} {
		res, err := call.endpoint()(ctx, call.value)
		require.NoError(t, err)
		require.Equal(t, call.value, res)
	}
	res, err := c.Describe()(ctx, "detail")
	require.NoError(t, err)
	require.Equal(t, "detail", *res.(*shapes.Detail).Name)
}

func TestEmptyObjects(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()

	res, err := c.Empty()(ctx, &shapes.Nothing{})
	require.NoError(t, err)
	require.Equal(t, &shapes.Nothing{}, res)

	res, err = c.Watch()(ctx, &shapes.Nothing{})
	require.NoError(t, err)
	watch := res.(*client.WatchClientStream)
	item, err := watch.Recv()
	require.NoError(t, err)
	require.Equal(t, &shapes.Nothing{}, item)
	_, err = watch.Recv()
	require.ErrorIs(t, err, io.EOF)

	res, err = c.Upload()(ctx, nil)
	require.NoError(t, err)
	upload := res.(*client.UploadClientStream)
	require.NoError(t, upload.Send(&shapes.Nothing{}))
	require.NoError(t, upload.Send(&shapes.Nothing{}))
	result, err := upload.CloseAndRecv()
	require.NoError(t, err)
	require.Equal(t, &shapes.Nothing{}, result)

	payload, err := client.BuildEmptyPayload()
	require.NoError(t, err)
	require.Equal(t, &shapes.Nothing{}, payload)
	id, err := client.BuildIDPayload(` + "`" + `{"field": "cli"}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, shapes.ID2("cli"), id)
}

func TestSharedErrorMessage(t *testing.T) {
	c := newClient(t)
	ctx := context.Background()
	for _, name := range []string{"missing", "invalid"} {
		t.Run(name, func(t *testing.T) {
			_, err := c.Fail()(ctx, name)
			var fault *shapes.Fault
			require.ErrorAs(t, err, &fault)
			require.Equal(t, name, fault.Name)
			require.Equal(t, "failed "+name, *fault.Msg)

			res, err := c.FailStream()(ctx, name)
			require.NoError(t, err)
			_, err = res.(*client.FailStreamClientStream).Recv()
			fault = nil
			require.ErrorAs(t, err, &fault)
			require.Equal(t, name, fault.Name)
			require.Equal(t, "failed "+name, *fault.Msg)
		})
	}
}
`
