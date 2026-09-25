package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNestedCollectionParamsGeneratedModule generates a unary and a
// WebSocket JSON-RPC service whose params are flat and nested collections of
// one element type, compiles and vets them, and round-trips the params
// through the generated clients and servers.
func TestJSONRPCNestedCollectionParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNestedCollectionParamsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnested", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested_test.go"), []byte(jsonRPCNestedCollectionParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcNestedCollectionParamsDSL() {
	API("nested", func() {
		JSONRPC(func() {})
	})
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	Service("unary", func() {
		JSONRPC(func() {
			POST("/unary")
		})
		Method("put", func() {
			Payload(ArrayOf(item))
			Result(Int)
			JSONRPC(func() {})
		})
		Method("put nested", func() {
			Payload(ArrayOf(ArrayOf(item)))
			Result(Int)
			JSONRPC(func() {})
		})
	})
	Service("stream", func() {
		JSONRPC(func() {
			GET("/stream")
		})
		Method("talk", func() {
			StreamingPayload(ArrayOf(item))
			Result(Int)
			JSONRPC(func() {})
		})
		Method("talk nested", func() {
			StreamingPayload(ArrayOf(ArrayOf(item)))
			Result(Int)
			JSONRPC(func() {})
		})
	})
}

const jsonRPCNestedCollectionParamsHarness = `package jsonrpcnested_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	streamclient "example.com/jsonrpcnested/gen/jsonrpc/stream/client"
	streamserver "example.com/jsonrpcnested/gen/jsonrpc/stream/server"
	unaryclient "example.com/jsonrpcnested/gen/jsonrpc/unary/client"
	unaryserver "example.com/jsonrpcnested/gen/jsonrpc/unary/server"
	stream "example.com/jsonrpcnested/gen/stream"
	unary "example.com/jsonrpcnested/gen/unary"
	loomhttp "github.com/CaliLuke/loom/http"
)

// names concatenates the names of the items so that the results prove
// which params the server decoded.
func names[T any](items []T, name func(T) string) int {
	n := 0
	for _, item := range items {
		n += len(name(item))
	}
	return n
}

type unaryService struct{}

func (unaryService) Put(_ context.Context, p []*unary.Item) (int, error) {
	return names(p, func(i *unary.Item) string { return i.Name }), nil
}

func (unaryService) PutNested(_ context.Context, p [][]*unary.Item) (int, error) {
	return names(p, func(l []*unary.Item) string {
		s := ""
		for _, i := range l {
			s += i.Name
		}
		return s
	}), nil
}

type streamService struct{}

func (streamService) HandleStream(ctx context.Context, s stream.Stream) error {
	for {
		if err := s.Recv(ctx); err != nil {
			return err
		}
	}
}

func (streamService) Talk(_ context.Context, p []*stream.Item) (int, error) {
	return names(p, func(i *stream.Item) string { return i.Name }), nil
}

func (streamService) TalkNested(_ context.Context, p [][]*stream.Item) (int, error) {
	return names(p, func(l []*stream.Item) string {
		s := ""
		for _, i := range l {
			s += i.Name
		}
		return s
	}), nil
}

func TestUnaryParams(t *testing.T) {
	mux := loomhttp.NewMuxer()
	unaryserver.Mount(mux, unaryserver.New(unary.NewEndpoints(unaryService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := unaryclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	if got, err := c.Put()(ctx, []*unary.Item{{Name: "ab"}, {Name: "c"}}); err != nil || got != 3 {
		t.Errorf("put: got %v, %v, want 3", got, err)
	}
	if got, err := c.PutNested()(ctx, [][]*unary.Item{{{Name: "ab"}}, {{Name: "c"}, {Name: "de"}}}); err != nil || got != 5 {
		t.Errorf("put nested: got %v, %v, want 5", got, err)
	}
	if _, err := c.PutNested()(ctx, [][]*unary.Item{{{Name: ""}}}); err == nil {
		t.Error("put nested accepted an empty name")
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

	res, err := c.Talk()(ctx, nil)
	if err != nil {
		t.Fatalf("talk: %v", err)
	}
	talk := res.(*streamclient.TalkClientStream)
	if err := talk.SendWithContext(ctx, []*stream.Item{{Name: "ab"}, {Name: "c"}}); err != nil {
		t.Fatalf("talk send: %v", err)
	}
	if got, err := talk.CloseAndRecvWithContext(ctx); err != nil || got != 3 {
		t.Errorf("talk: got %v, %v, want 3", got, err)
	}
	if err := talk.Close(); err != nil {
		t.Errorf("talk close: %v", err)
	}

	res, err = c.TalkNested()(ctx, nil)
	if err != nil {
		t.Fatalf("talk nested: %v", err)
	}
	nested := res.(*streamclient.TalkNestedClientStream)
	if err := nested.SendWithContext(ctx, [][]*stream.Item{{{Name: "ab"}}, {{Name: "c"}, {Name: "de"}}}); err != nil {
		t.Fatalf("talk nested send: %v", err)
	}
	if got, err := nested.CloseAndRecvWithContext(ctx); err != nil || got != 5 {
		t.Errorf("talk nested: got %v, %v, want 5", got, err)
	}
	if err := nested.Close(); err != nil {
		t.Errorf("talk nested close: %v", err)
	}
}
`
