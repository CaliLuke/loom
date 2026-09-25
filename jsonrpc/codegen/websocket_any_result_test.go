package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCWebSocketAnyResultGeneratedModule covers the JSON-RPC WebSocket
// client streams of methods whose result, streaming result or streaming
// payload is Any, an array of Any or a type over Any. The client refers to
// loom.JSONValue and imports the loom package. JSON values of every kind go
// through the connection unchanged.
func TestJSONRPCWebSocketAnyResultGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketAnyResultDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsany", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "any_test.go"), []byte(jsonRPCWebSocketAnyResultHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

// TestJSONRPCWebSocketAnyResultLoomServiceGeneratedModule covers a service
// named loom, like the import of the loom package that the client of an Any
// result needs. The client imports the service package under an alias.
func TestJSONRPCWebSocketAnyResultLoomServiceGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("wsanyloom", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("loom", func() {
			dsl.JSONRPC(func() {
				dsl.GET("/ws")
			})
			dsl.Method("upload", func() {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(dsl.Any)
				dsl.JSONRPC(func() {})
			})
			dsl.Method("echo", func() {
				dsl.StreamingPayload(dsl.Any)
				dsl.StreamingResult(dsl.Any)
				dsl.JSONRPC(func() {})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsanyloom", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}

func jsonrpcWebSocketAnyResultDSL() {
	dsl.API("wsany", func() {
		dsl.JSONRPC(func() {})
	})
	blob := dsl.Type("Blob", dsl.Any)
	dsl.Service("store", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.Any)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("uploadBlob", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(blob)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("uploadList", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.ArrayOf(dsl.Any))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("echo", func() {
			dsl.StreamingPayload(dsl.Any)
			dsl.StreamingResult(dsl.Any)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("watch", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(blob)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCWebSocketAnyResultHarness = `package jsonrpcwsany_test

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	client "example.com/jsonrpcwsany/gen/jsonrpc/store/client"
	server "example.com/jsonrpcwsany/gen/jsonrpc/store/server"
	store "example.com/jsonrpcwsany/gen/store"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// values holds a JSON value of every kind; each string payload names the
// index of the value that the methods answer with.
var values = []loom.JSONValue{
	loom.JSONValue(` + "`" + `{"k":[1,"x",null],"n":{}}` + "`" + `),
	loom.JSONValue(` + "`" + `"\r\n"` + "`" + `),
	loom.JSONValue(` + "`" + `1.5` + "`" + `),
	loom.JSONValue(` + "`" + `[]` + "`" + `),
	loom.JSONValue(` + "`" + `true` + "`" + `),
	loom.JSONValue(` + "`" + `null` + "`" + `),
}

var payloads = []string{"0", "1", "2", "3", "4", "5"}

func pick(p string) loom.JSONValue {
	return values[p[0]-'0']
}

type service struct{}

func (service) HandleStream(ctx context.Context, stream store.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (service) Upload(_ context.Context, p string) (loom.JSONValue, error) {
	return pick(p), nil
}

func (service) UploadBlob(_ context.Context, p string) (store.Blob, error) {
	return pick(p), nil
}

func (service) UploadList(_ context.Context, p string) ([]loom.JSONValue, error) {
	return []loom.JSONValue{pick(p), pick(p)}, nil
}

func (service) Echo(ctx context.Context, p loom.JSONValue, st store.EchoServerStream) error {
	return st.SendResponse(ctx, p)
}

func (service) Watch(ctx context.Context, p string, st store.WatchServerStream) error {
	return st.SendResponse(ctx, pick(p))
}

func newClient(t *testing.T) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	svc := service{}
	server.Mount(mux, server.New(svc.HandleStream, store.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	return c
}

// exchange sends every payload on a stream opened with open, then receives
// the response to each one with recv and checks it against want.
func exchange[S interface{ Close() error }, P, R any](t *testing.T, name string, open func(context.Context, any) (any, error), send func(S, context.Context, P) error, recv func(S, context.Context) (R, error), payloads []P, want func(int) R) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := open(ctx, nil)
	if err != nil {
		t.Fatalf("%s: open: %v", name, err)
	}
	stream := res.(S)
	for _, p := range payloads {
		if err := send(stream, ctx, p); err != nil {
			t.Fatalf("%s: send %v: %v", name, p, err)
		}
	}
	for i := range payloads {
		got, err := recv(stream, ctx)
		if err != nil || !reflect.DeepEqual(got, want(i)) {
			t.Errorf("%s: recv %d: got (%v, %v), want %v", name, i, got, err, want(i))
		}
	}
	if err := stream.Close(); err != nil {
		t.Errorf("%s: close: %v", name, err)
	}
}

func TestAnyResults(t *testing.T) {
	c := newClient(t)
	value := func(i int) loom.JSONValue { return values[i] }
	exchange(t, "upload", c.Upload(), (*client.UploadClientStream).SendWithContext, (*client.UploadClientStream).CloseAndRecvWithContext, payloads, value)
	exchange(t, "uploadBlob", c.UploadBlob(), (*client.UploadBlobClientStream).SendWithContext, (*client.UploadBlobClientStream).CloseAndRecvWithContext, payloads, func(i int) store.Blob { return values[i] })
	exchange(t, "uploadList", c.UploadList(), (*client.UploadListClientStream).SendWithContext, (*client.UploadListClientStream).CloseAndRecvWithContext, payloads, func(i int) []loom.JSONValue { return []loom.JSONValue{values[i], values[i]} })
	exchange(t, "echo", c.Echo(), (*client.EchoClientStream).SendWithContext, (*client.EchoClientStream).RecvWithContext, values, value)
	exchange(t, "watch", c.Watch(), (*client.WatchClientStream).SendWithContext, (*client.WatchClientStream).RecvWithContext, payloads, func(i int) store.Blob { return values[i] })
}
`
