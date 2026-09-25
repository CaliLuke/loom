package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestNamedCollectionPayloadGeneratedIntegration generates a service whose
// unary and WebSocket streaming payloads are two named collections of one
// element type, and whose request bodies are two collection attributes of
// one payload type. It compiles and vets the service, then checks that the
// generated client sends each payload, and each selected attribute, to the
// server, and that the server rejects a payload that violates the
// validations of its type.
func TestNamedCollectionPayloadGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, namedCollectionPayloadDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/namedcollection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "named_collection_test.go"), []byte(namedCollectionPayloadHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func namedCollectionPayloadDSL() {
	item := Type("Item", func() {
		Attribute("name", String)
		Required("name")
	})
	l1 := Type("L1", ArrayOf(item))
	l2 := Type("L2", ArrayOf(item), func() {
		MinLength(1)
	})
	pair := Type("Pair", func() {
		Attribute("a", ArrayOf(item))
		Attribute("b", ArrayOf(item))
		Required("a", "b")
	})
	Service("named", func() {
		unary := func(name, path string, payload any, body string) {
			Method(name, func() {
				Payload(payload)
				Result(String)
				HTTP(func() {
					POST(path)
					if body != "" {
						Body(body)
					}
				})
			})
		}
		stream := func(name, path string, payload any) {
			Method(name, func() {
				StreamingPayload(payload)
				Result(String)
				HTTP(func() {
					GET(path)
				})
			})
		}
		unary("one", "/one", l1, "")
		unary("two", "/two", l2, "")
		stream("stream one", "/stream-one", l1)
		stream("stream two", "/stream-two", l2)
		unary("first", "/first", pair, "a")
		unary("second", "/second", pair, "b")
	})
}

const namedCollectionPayloadHarness = `package namedcollection

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	client "example.com/namedcollection/gen/http/named/client"
	server "example.com/namedcollection/gen/http/named/server"
	named "example.com/namedcollection/gen/named"
	loomhttp "github.com/CaliLuke/loom/http"
)

func join(items []*named.Item) string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return strings.Join(names, ",")
}

type recv[T any] interface {
	RecvWithContext(context.Context) (T, error)
	SendAndCloseWithContext(context.Context, string) error
}

func collect[T ~[]*named.Item](ctx context.Context, stream recv[T]) error {
	var all []string
	for {
		msg, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return stream.SendAndCloseWithContext(ctx, strings.Join(all, ";"))
		}
		if err != nil {
			return err
		}
		all = append(all, join(msg))
	}
}

type service struct{}

func (service) One(_ context.Context, p named.L1) (string, error) {
	return "one:" + join(p), nil
}

func (service) Two(_ context.Context, p named.L2) (string, error) {
	return "two:" + join(p), nil
}

func (service) StreamOne(ctx context.Context, stream named.StreamOneServerStream) error {
	return collect[named.L1](ctx, stream)
}

func (service) StreamTwo(ctx context.Context, stream named.StreamTwoServerStream) error {
	return collect[named.L2](ctx, stream)
}

func (service) First(_ context.Context, p *named.Pair) (string, error) {
	return "first:" + join(p.A), nil
}

func (service) Second(_ context.Context, p *named.Pair) (string, error) {
	return "second:" + join(p.B), nil
}

func serve(t *testing.T) *httptest.Server {
	t.Helper()
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(named.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs
}

func newClient(hs *httptest.Server, scheme string) *client.Client {
	return client.NewClient(scheme, strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
}

func TestUnary(t *testing.T) {
	c := newClient(serve(t), "http")
	ctx := context.Background()
	pair := &named.Pair{A: []*named.Item{{Name: "a"}}, B: []*named.Item{{Name: "b1"}, {Name: "b2"}}}
	cases := []struct {
		name     string
		endpoint func(context.Context, any) (any, error)
		payload  any
		want     string
	}{
		{"one", c.One(), named.L1{{Name: "x"}, {Name: "y"}}, "one:x,y"},
		{"two", c.Two(), named.L2{{Name: "z"}}, "two:z"},
		{"first", c.First(), pair, "first:a"},
		{"second", c.Second(), pair, "second:b1,b2"},
	}
	for _, tc := range cases {
		if got, err := tc.endpoint(ctx, tc.payload); err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v, want %q", tc.name, got, err, tc.want)
		}
	}
}

func TestStream(t *testing.T) {
	c := newClient(serve(t), "ws")
	ctx := context.Background()

	raw, err := c.StreamOne()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream one: %v", err)
	}
	one := raw.(*client.StreamOneClientStream)
	for _, msg := range []named.L1{{{Name: "a"}}, {{Name: "b"}, {Name: "c"}}} {
		if err := one.Send(msg); err != nil {
			t.Fatalf("stream one send: %v", err)
		}
	}
	if got, err := one.CloseAndRecv(); err != nil || got != "a;b,c" {
		t.Errorf("stream one: got %q, %v, want %q", got, err, "a;b,c")
	}

	raw, err = c.StreamTwo()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream two: %v", err)
	}
	two := raw.(*client.StreamTwoClientStream)
	if err := two.Send(named.L2{{Name: "d"}}); err != nil {
		t.Fatalf("stream two send: %v", err)
	}
	if got, err := two.CloseAndRecv(); err != nil || got != "d" {
		t.Errorf("stream two: got %q, %v, want %q", got, err, "d")
	}
}

func TestRejectsEmptyL2(t *testing.T) {
	hs := serve(t)
	resp, err := http.Post(hs.URL+"/two", "application/json", strings.NewReader("[]"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", resp.StatusCode)
	}
	resp, err = http.Post(hs.URL+"/one", "application/json", strings.NewReader("[]"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("one status %d, want 200", resp.StatusCode)
	}
}
`
