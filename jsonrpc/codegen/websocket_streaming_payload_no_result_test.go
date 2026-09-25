package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCWebSocketStreamingPayloadNoResultGeneratedModule covers a
// JSON-RPC WebSocket method that declares only a streaming payload. The
// generated client sends each message as a notification, which the server
// handles without a response. A request with an ID gets a success response
// without a result, and a failing request with an ID gets the error.
func TestJSONRPCWebSocketStreamingPayloadNoResultGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketStreamingPayloadNoResultDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwssink", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sink_test.go"), []byte(jsonRPCWebSocketStreamingPayloadNoResultHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

func jsonrpcWebSocketStreamingPayloadNoResultDSL() {
	dsl.API("wssink", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("sink", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("push", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCWebSocketStreamingPayloadNoResultHarness = `package jsonrpcwssink_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	client "example.com/jsonrpcwssink/gen/jsonrpc/sink/client"
	server "example.com/jsonrpcwssink/gen/jsonrpc/sink/server"
	sink "example.com/jsonrpcwssink/gen/sink"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type service struct {
	pushed chan string
}

func (s *service) HandleStream(ctx context.Context, stream sink.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (s *service) Push(_ context.Context, p string) error {
	if p == "fail" {
		return loom.PermanentError("rejected", "push rejected")
	}
	s.pushed <- p
	return nil
}

func serve(t *testing.T) (*service, string) {
	t.Helper()
	svc := &service{pushed: make(chan string, 8)}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.HandleStream, sink.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return svc, strings.TrimPrefix(hs.URL, "http://")
}

func TestPushNotifications(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc, host := serve(t)
	c := client.NewClient("ws", host, nil, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	defer func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	}()
	res, err := c.Push()(ctx, nil)
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	stream := res.(*client.PushClientStream)
	for _, v := range []string{"a", "b"} {
		if err := stream.SendWithContext(ctx, v); err != nil {
			t.Fatalf("send %s: %v", v, err)
		}
	}
	for _, want := range []string{"a", "b"} {
		select {
		case got := <-svc.pushed:
			if got != want {
				t.Errorf("pushed %q, want %q", got, want)
			}
		case <-ctx.Done():
			t.Fatalf("service did not receive %q", want)
		}
	}
	if err := stream.Close(); err != nil {
		t.Errorf("close stream: %v", err)
	}
}

type response struct {
	JSONRPC string          ` + "`json:\"jsonrpc\"`" + `
	ID      any             ` + "`json:\"id\"`" + `
	Result  jsontext.Value  ` + "`json:\"result\"`" + `
	Error   *struct {
		Code    int    ` + "`json:\"code\"`" + `
		Message string ` + "`json:\"message\"`" + `
	} ` + "`json:\"error\"`" + `
}

func TestPushRequestWithID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc, host := serve(t)
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws://"+host+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close handshake body: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, websocket.ErrCloseSent) {
			t.Errorf("close conn: %v", err)
		}
	}()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	cases := []struct {
		id      int
		params  string
		wantErr bool
	}{
		{1, "hello", false},
		{2, "fail", true},
	}
	for _, c := range cases {
		req := fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"method":"push","params":%q}` + "`" + `, c.id, c.params)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(req)); err != nil {
			t.Fatalf("write %d: %v", c.id, err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", c.id, err)
		}
		var got response
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("decode %d: %v: %s", c.id, err, data)
		}
		if got.JSONRPC != "2.0" || got.ID != float64(c.id) {
			t.Errorf("response %d: got %s", c.id, data)
		}
		if c.wantErr {
			if got.Error == nil || got.Error.Message == "" {
				t.Errorf("response %d: want an error, got %s", c.id, data)
			}
			continue
		}
		if got.Error != nil || len(got.Result) != 0 && string(got.Result) != "null" {
			t.Errorf("response %d: want success without result, got %s", c.id, data)
		}
	}
	select {
	case got := <-svc.pushed:
		if got != "hello" {
			t.Errorf("pushed %q, want hello", got)
		}
	default:
		t.Error("service did not receive hello")
	}
}
`
