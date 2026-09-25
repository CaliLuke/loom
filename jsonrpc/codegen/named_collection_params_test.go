package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNamedCollectionParamsGeneratedModule generates a unary and a
// WebSocket JSON-RPC service whose params are two named collections of one
// element type, compiles and vets them, and round-trips the params through
// the generated clients and servers.
func TestJSONRPCNamedCollectionParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNamedCollectionParamsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnamed", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "named_test.go"), []byte(jsonRPCNamedCollectionParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcNamedCollectionParamsDSL() {
	API("named", func() {
		JSONRPC(func() {})
	})
	item := Type("Item", func() {
		Attribute("name", String)
		Required("name")
	})
	l1 := Type("L1", ArrayOf(item))
	l2 := Type("L2", ArrayOf(item), func() {
		MinLength(1)
	})
	Service("unary", func() {
		JSONRPC(func() {
			POST("/unary")
		})
		Method("one", func() {
			Payload(l1)
			Result(String)
			JSONRPC(func() {})
		})
		Method("two", func() {
			Payload(l2)
			Result(String)
			JSONRPC(func() {})
		})
	})
	Service("stream", func() {
		JSONRPC(func() {
			GET("/stream")
		})
		Method("one", func() {
			StreamingPayload(l1)
			Result(String)
			JSONRPC(func() {})
		})
		Method("two", func() {
			StreamingPayload(l2)
			Result(String)
			JSONRPC(func() {})
		})
	})
}

const jsonRPCNamedCollectionParamsHarness = `package jsonrpcnamed_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	streamclient "example.com/jsonrpcnamed/gen/jsonrpc/stream/client"
	streamserver "example.com/jsonrpcnamed/gen/jsonrpc/stream/server"
	unaryclient "example.com/jsonrpcnamed/gen/jsonrpc/unary/client"
	unaryserver "example.com/jsonrpcnamed/gen/jsonrpc/unary/server"
	stream "example.com/jsonrpcnamed/gen/stream"
	unary "example.com/jsonrpcnamed/gen/unary"
	loomhttp "github.com/CaliLuke/loom/http"
)

func unaryNames(items []*unary.Item) string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

func streamNames(items []*stream.Item) string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

type unaryService struct{}

func (unaryService) One(_ context.Context, p unary.L1) (string, error) {
	return "one:" + unaryNames(p), nil
}

func (unaryService) Two(_ context.Context, p unary.L2) (string, error) {
	return "two:" + unaryNames(p), nil
}

type streamService struct{}

func (streamService) HandleStream(ctx context.Context, s stream.Stream) error {
	for {
		if err := s.Recv(ctx); err != nil {
			return err
		}
	}
}

func (streamService) One(_ context.Context, p stream.L1) (string, error) {
	return "one:" + streamNames(p), nil
}

func (streamService) Two(_ context.Context, p stream.L2) (string, error) {
	return "two:" + streamNames(p), nil
}

func TestUnaryParams(t *testing.T) {
	mux := loomhttp.NewMuxer()
	unaryserver.Mount(mux, unaryserver.New(unary.NewEndpoints(unaryService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := unaryclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	if got, err := c.One()(ctx, unary.L1{{Name: "a"}, {Name: "b"}}); err != nil || got != "one:a,b" {
		t.Errorf("one: got %v, %v, want one:a,b", got, err)
	}
	if got, err := c.Two()(ctx, unary.L2{{Name: "c"}}); err != nil || got != "two:c" {
		t.Errorf("two: got %v, %v, want two:c", got, err)
	}
	if _, err := c.Two()(ctx, unary.L2{}); err == nil {
		t.Error("two accepted an empty list")
	}
}

func TestStreamParams(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mux := loomhttp.NewMuxer()
	svc := streamService{}
	streamserver.Mount(mux, streamserver.New(svc.HandleStream, stream.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := streamclient.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	defer func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	}()

	res, err := c.One()(ctx, nil)
	if err != nil {
		t.Fatalf("one: %v", err)
	}
	one := res.(*streamclient.OneClientStream)
	if err := one.SendWithContext(ctx, stream.L1{{Name: "a"}, {Name: "b"}}); err != nil {
		t.Fatalf("one send: %v", err)
	}
	if got, err := one.CloseAndRecvWithContext(ctx); err != nil || got != "one:a,b" {
		t.Errorf("one: got %v, %v, want one:a,b", got, err)
	}
	if err := one.Close(); err != nil {
		t.Errorf("one close: %v", err)
	}

	res, err = c.Two()(ctx, nil)
	if err != nil {
		t.Fatalf("two: %v", err)
	}
	two := res.(*streamclient.TwoClientStream)
	if err := two.SendWithContext(ctx, stream.L2{{Name: "c"}}); err != nil {
		t.Fatalf("two send: %v", err)
	}
	if got, err := two.CloseAndRecvWithContext(ctx); err != nil || got != "two:c" {
		t.Errorf("two: got %v, %v, want two:c", got, err)
	}
	if err := two.Close(); err != nil {
		t.Errorf("two close: %v", err)
	}
}
`
