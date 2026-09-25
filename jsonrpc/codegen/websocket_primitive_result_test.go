package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCWebSocketPrimitiveResultGeneratedModule covers the JSON-RPC
// WebSocket client streams of methods whose result or streaming result is a
// primitive or a named primitive. A failed receive returns the zero value of
// the type, which is not nil, and the zero values of the types go through the
// connection as results.
func TestJSONRPCWebSocketPrimitiveResultGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPrimitiveResultDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsprimitive", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "primitive_test.go"), []byte(jsonRPCWebSocketPrimitiveResultHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

// TestJSONRPCWebSocketResultPackageShadowingGeneratedModule covers services
// named after the locals of the generated client stream functions, out and
// zero. Their result types are qualified with the service package, which the
// locals must not shadow where the functions refer to the result type.
func TestJSONRPCWebSocketResultPackageShadowingGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketResultPackageShadowingDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsshadow", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}

func jsonrpcWebSocketResultPackageShadowingDSL() {
	dsl.API("wsshadow", func() {
		dsl.JSONRPC(func() {})
	})
	frame := dsl.Type("Frame", func() {
		dsl.Attribute("text", dsl.String)
	})
	for _, name := range []string{"out", "zero"} {
		dsl.Service(name, func() {
			dsl.JSONRPC(func() {
				dsl.GET("/" + name)
			})
			dsl.Method("talk", func() {
				dsl.StreamingPayload(frame)
				dsl.StreamingResult(frame)
				dsl.JSONRPC(func() {})
			})
			dsl.Method("upload", func() {
				dsl.StreamingPayload(frame)
				dsl.Result(frame)
				dsl.JSONRPC(func() {})
			})
		})
	}
}

func jsonrpcWebSocketPrimitiveResultDSL() {
	dsl.API("wsprimitive", func() {
		dsl.JSONRPC(func() {})
	})
	token := dsl.Type("Token", dsl.String)
	count := dsl.Type("Count", dsl.Int)
	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("length", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.Int)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("tally", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(count)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("echo", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("tokens", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(token)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("flags", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(dsl.Boolean)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("ticks", func() {
			dsl.Payload(dsl.Int)
			dsl.StreamingResult(count)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCWebSocketPrimitiveResultHarness = `package jsonrpcwsprimitive_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	calc "example.com/jsonrpcwsprimitive/gen/calc"
	client "example.com/jsonrpcwsprimitive/gen/jsonrpc/calc/client"
	server "example.com/jsonrpcwsprimitive/gen/jsonrpc/calc/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// service fails every payload "fail" and otherwise answers with a value
// derived from the payload; the empty payload yields the zero value.
type service struct{}

func (service) HandleStream(ctx context.Context, stream calc.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func rejected() error {
	return loom.PermanentError("rejected", "payload rejected")
}

func (service) Length(_ context.Context, p string) (int, error) {
	if p == "fail" {
		return 0, rejected()
	}
	return len(p), nil
}

func (service) Tally(_ context.Context, p string) (calc.Count, error) {
	if p == "fail" {
		return 0, rejected()
	}
	return calc.Count(len(p)), nil
}

func (service) Echo(ctx context.Context, p string, st calc.EchoServerStream) error {
	if p == "fail" {
		return rejected()
	}
	return st.SendResponse(ctx, p)
}

func (service) Tokens(ctx context.Context, p string, st calc.TokensServerStream) error {
	if p == "fail" {
		return rejected()
	}
	return st.SendResponse(ctx, calc.Token(p))
}

func (service) Flags(ctx context.Context, p string, st calc.FlagsServerStream) error {
	if p == "fail" {
		return rejected()
	}
	return st.SendResponse(ctx, p != "")
}

func (service) Ticks(ctx context.Context, p int, st calc.TicksServerStream) error {
	if p < 0 {
		return rejected()
	}
	return st.SendResponse(ctx, calc.Count(p*2))
}

func newClient(t *testing.T) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	svc := service{}
	server.Mount(mux, server.New(svc.HandleStream, calc.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
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

// exchange sends every payload on a stream opened with open and receives the
// response to each one with recv. It checks that the results are want and
// that a payload "fail" gets an error with the zero value.
func exchange[P any, R comparable](t *testing.T, name string, open func() (any, error), send func(any, context.Context, P) error, recv func(any, context.Context) (R, error), closeStream func(any) error, payloads []P, want []R, fail P) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := open()
	if err != nil {
		t.Fatalf("%s: open: %v", name, err)
	}
	for _, p := range append(payloads, fail) {
		if err := send(stream, ctx, p); err != nil {
			t.Fatalf("%s: send %v: %v", name, p, err)
		}
	}
	for i, w := range want {
		got, err := recv(stream, ctx)
		if err != nil || got != w {
			t.Errorf("%s: recv %d: got (%v, %v), want %v", name, i, got, err, w)
		}
	}
	var zero R
	if got, err := recv(stream, ctx); err == nil || got != zero {
		t.Errorf("%s: recv fail: got (%v, %v), want the zero value and an error", name, got, err)
	}
	if err := closeStream(stream); err != nil {
		t.Errorf("%s: close: %v", name, err)
	}
}

func TestPrimitiveResults(t *testing.T) {
	c := newClient(t)
	open := func(e loom.Endpoint) func() (any, error) {
		return func() (any, error) {
			return e(context.Background(), nil)
		}
	}
	closeStream := func(s any) error {
		return s.(interface{ Close() error }).Close()
	}
	strs := []string{"abc", ""}

	exchange(t, "length", open(c.Length()),
		func(s any, ctx context.Context, p string) error { return s.(*client.LengthClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (int, error) { return s.(*client.LengthClientStream).CloseAndRecvWithContext(ctx) },
		closeStream, strs, []int{3, 0}, "fail")
	exchange(t, "tally", open(c.Tally()),
		func(s any, ctx context.Context, p string) error { return s.(*client.TallyClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (calc.Count, error) { return s.(*client.TallyClientStream).CloseAndRecvWithContext(ctx) },
		closeStream, strs, []calc.Count{3, 0}, "fail")
	exchange(t, "echo", open(c.Echo()),
		func(s any, ctx context.Context, p string) error { return s.(*client.EchoClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (string, error) { return s.(*client.EchoClientStream).RecvWithContext(ctx) },
		closeStream, strs, []string{"abc", ""}, "fail")
	exchange(t, "tokens", open(c.Tokens()),
		func(s any, ctx context.Context, p string) error { return s.(*client.TokensClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (calc.Token, error) { return s.(*client.TokensClientStream).RecvWithContext(ctx) },
		closeStream, strs, []calc.Token{"abc", ""}, "fail")
	exchange(t, "flags", open(c.Flags()),
		func(s any, ctx context.Context, p string) error { return s.(*client.FlagsClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (bool, error) { return s.(*client.FlagsClientStream).RecvWithContext(ctx) },
		closeStream, strs, []bool{true, false}, "fail")
	exchange(t, "ticks", open(c.Ticks()),
		func(s any, ctx context.Context, p int) error { return s.(*client.TicksClientStream).SendWithContext(ctx, p) },
		func(s any, ctx context.Context) (calc.Count, error) { return s.(*client.TicksClientStream).RecvWithContext(ctx) },
		closeStream, []int{2, 0}, []calc.Count{4, 0}, -1)
}

// TestPrimitiveResultCanceled checks that a receive that ends with its
// context returns the zero value.
func TestPrimitiveResultCanceled(t *testing.T) {
	c := newClient(t)
	res, err := c.Length()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	stream := res.(*client.LengthClientStream)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := stream.CloseAndRecvWithContext(ctx); err == nil || got != 0 {
		t.Errorf("recv: got (%v, %v), want 0 and an error", got, err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}
`
