package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCSSEMixedResultsGeneratedModuleRoundTrip covers JSON-RPC SSE
// methods whose unary result and streaming result differ. The event type
// follows the streaming result: a built-in streaming result is an alias even
// when the unary result is a local user type, and a local streaming result
// keeps its marker even when the unary result is built in. Server and client
// must both use the streaming result type. The service also streams Any and
// an Any-based user type, which are the same Go type, so the service-level
// Send type switch must hold a single case for them.
func TestJSONRPCSSEMixedResultsGeneratedModuleRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-sse-mixed", func() {
			dsl.JSONRPC(func() {})
		})
		note := dsl.Type("Note", func() {
			dsl.Attribute("text", dsl.String)
			dsl.Required("text")
		})
		item := dsl.Type("Item", func() {
			dsl.Attribute("id", dsl.Int)
			dsl.Required("id")
		})
		doc := dsl.Type("Doc", dsl.Any)
		dsl.Service("Mixed", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			mixed := func(name string, result, streaming any) {
				dsl.Method(name, func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
					})
					dsl.Result(result)
					dsl.StreamingResult(streaming)
					dsl.JSONRPC(func() {
						dsl.ServerSentEvents()
					})
				})
			}
			mixed("LocalResult", note, dsl.String)
			mixed("LocalStream", dsl.String, note)
			mixed("LocalBoth", note, item)
			stream := func(name string, result any) {
				dsl.Method(name, func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
					})
					dsl.StreamingResult(result)
					dsl.JSONRPC(func() {
						dsl.ServerSentEvents()
					})
				})
			}
			stream("StreamAny", dsl.Any)
			stream("StreamDoc", doc)
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcssemixed", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mixed_test.go"), []byte(jsonRPCSSEMixedHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const jsonRPCSSEMixedHarness = `package jsonrpcssemixed_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	client "example.com/jsonrpcssemixed/gen/jsonrpc/mixed/client"
	server "example.com/jsonrpcssemixed/gen/jsonrpc/mixed/server"
	svc "example.com/jsonrpcssemixed/gen/mixed"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

var (
	stringValues = []string{"", "\r\n", "data: fake\n\nevent: x", "last"}
	noteValues   = []*svc.Note{{Text: ""}, {Text: "a\r\nb\n"}}
	itemValues   = []*svc.Item{{ID: 0}, {ID: -7}}
	anyValues    = []loom.JSONValue{loom.JSONValue("{\"k\":[1,null]}"), loom.JSONValue("\"\\r\\n\"")}
	docValues    = []svc.Doc{svc.Doc("\"\""), svc.Doc("[\"\\n\"]")}
)

func send[T any](ctx context.Context, send, final func(context.Context, T) error, values []T) error {
	for _, v := range values[:len(values)-1] {
		if err := send(ctx, v); err != nil {
			return err
		}
	}
	return final(ctx, values[len(values)-1])
}

func (service) LocalBoth(ctx context.Context, _ *svc.LocalBothPayload, st svc.LocalBothServerStream) (*svc.Note, error) {
	s := func(ctx context.Context, v *svc.Item) error { return st.Send(ctx, v) }
	f := func(ctx context.Context, v *svc.Item) error { return st.SendAndClose(ctx, v) }
	return nil, send(ctx, s, f, itemValues)
}

func (service) StreamAny(ctx context.Context, _ *svc.StreamAnyPayload, st svc.StreamAnyServerStream) error {
	return send(ctx, st.Send, st.SendAndClose, anyValues)
}

func (service) StreamDoc(ctx context.Context, _ *svc.StreamDocPayload, st svc.StreamDocServerStream) error {
	return send(ctx, st.Send, st.SendAndClose, docValues)
}

type service struct{}

func (service) LocalResult(ctx context.Context, _ *svc.LocalResultPayload, st svc.LocalResultServerStream) (*svc.Note, error) {
	for _, v := range stringValues[:len(stringValues)-1] {
		if err := st.Send(ctx, v); err != nil {
			return nil, err
		}
	}
	return nil, st.SendAndClose(ctx, stringValues[len(stringValues)-1])
}

func (service) LocalStream(ctx context.Context, _ *svc.LocalStreamPayload, st svc.LocalStreamServerStream) (string, error) {
	for _, v := range noteValues[:len(noteValues)-1] {
		if err := st.Send(ctx, v); err != nil {
			return "", err
		}
	}
	return "", st.SendAndClose(ctx, noteValues[len(noteValues)-1])
}

type recvStream[T any] interface {
	Recv(context.Context) (T, error)
	Close() error
}

func requireRoundTrip[T any](t *testing.T, name string, endpoint loom.Endpoint, payload any, want []T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := endpoint(ctx, payload)
	if err != nil {
		t.Fatalf("%s: connect: %v", name, err)
	}
	s, ok := res.(recvStream[T])
	if !ok {
		t.Fatalf("%s: unexpected stream type %T", name, res)
	}
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("%s: close: %v", name, err)
		}
	}()
	for i, w := range want {
		got, err := s.Recv(ctx)
		if err != nil {
			t.Errorf("%s[%d]: recv: %v", name, i, err)
			return
		}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("%s[%d]: got %#v, want %#v", name, i, got, w)
		}
	}
	if _, err := s.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("%s: expected io.EOF after %d events, got %v", name, len(want), err)
	}
}

func TestMixedResultsRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	srv := server.New(svc.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler)
	server.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	id := "req-1"
	requireRoundTrip(t, "local result", c.LocalResult(), &svc.LocalResultPayload{ID: &id}, stringValues)
	requireRoundTrip(t, "local stream", c.LocalStream(), &svc.LocalStreamPayload{ID: &id}, noteValues)
	requireRoundTrip(t, "local both", c.LocalBoth(), &svc.LocalBothPayload{ID: &id}, itemValues)
	requireRoundTrip(t, "any", c.StreamAny(), &svc.StreamAnyPayload{ID: &id}, anyValues)
	requireRoundTrip(t, "doc", c.StreamDoc(), &svc.StreamDocPayload{ID: &id}, docValues)
}
`
