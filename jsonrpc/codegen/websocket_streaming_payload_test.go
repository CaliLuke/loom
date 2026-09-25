package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCWebSocketStreamingPayloadGeneratedModule covers the methods of a
// JSON-RPC WebSocket service that receive a streaming payload and return a
// result. The server handles each message as a request and must send the
// result type of the method, here a user type, an inline object and an
// array, as the response to the request.
func TestJSONRPCWebSocketStreamingPayloadGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketStreamingPayloadDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwspayload", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "payload_test.go"), []byte(jsonRPCWebSocketStreamingPayloadHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

func jsonrpcWebSocketStreamingPayloadDSL() {
	dsl.API("wspayload", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.Type("Note", func() {
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	dsl.Service("files", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(note)
			dsl.Result(note)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("count", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(func() {
				dsl.Attribute("count", dsl.Int)
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("names", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.ArrayOf(dsl.String))
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCWebSocketStreamingPayloadHarness = `package jsonrpcwspayload_test

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	files "example.com/jsonrpcwspayload/gen/files"
	client "example.com/jsonrpcwspayload/gen/jsonrpc/files/client"
	server "example.com/jsonrpcwspayload/gen/jsonrpc/files/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct{}

func (service) HandleStream(ctx context.Context, stream files.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (service) Upload(_ context.Context, p *files.Note) (*files.Note, error) {
	return &files.Note{Text: p.Text + "!"}, nil
}

func (service) Count(_ context.Context, p string) (*files.CountResult, error) {
	n := len(p)
	return &files.CountResult{Count: &n}, nil
}

func (service) Names(_ context.Context, p string) ([]string, error) {
	return strings.Split(p, ","), nil
}

func newClient(t *testing.T) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	svc := service{}
	server.Mount(mux, server.New(svc.HandleStream, files.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
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

func TestStreamingPayloadResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := newClient(t)

	res, err := c.Upload()(ctx, nil)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	upload := res.(*client.UploadClientStream)
	for _, text := range []string{"a", "b"} {
		if err := upload.SendWithContext(ctx, &files.Note{Text: text}); err != nil {
			t.Fatalf("upload send %s: %v", text, err)
		}
	}
	for _, want := range []string{"a!", "b!"} {
		got, err := upload.CloseAndRecvWithContext(ctx)
		if err != nil || got == nil || got.Text != want {
			t.Errorf("upload: got (%+v, %v), want %s", got, err, want)
		}
	}
	if err := upload.Close(); err != nil {
		t.Errorf("upload close: %v", err)
	}

	res, err = c.Count()(ctx, nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	count := res.(*client.CountClientStream)
	if err := count.SendWithContext(ctx, "abc"); err != nil {
		t.Fatalf("count send: %v", err)
	}
	if got, err := count.CloseAndRecvWithContext(ctx); err != nil || got == nil || got.Count == nil || *got.Count != 3 {
		t.Errorf("count: got (%+v, %v), want 3", got, err)
	}
	if err := count.Close(); err != nil {
		t.Errorf("count close: %v", err)
	}

	res, err = c.Names()(ctx, nil)
	if err != nil {
		t.Fatalf("names: %v", err)
	}
	names := res.(*client.NamesClientStream)
	if err := names.SendWithContext(ctx, "x,y"); err != nil {
		t.Fatalf("names send: %v", err)
	}
	if got, err := names.CloseAndRecvWithContext(ctx); err != nil || !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Errorf("names: got (%v, %v), want [x y]", got, err)
	}
	if err := names.Close(); err != nil {
		t.Errorf("names close: %v", err)
	}
}
`
