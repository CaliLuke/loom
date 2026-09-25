package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestNestedCollectionPayloadGeneratedIntegration generates a service whose
// unary and WebSocket streaming payloads are flat and nested collections of
// one element type, compiles and vets it, then round-trips the payloads
// through the generated client and server and checks that the server
// rejects invalid nested elements.
func TestNestedCollectionPayloadGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, nestedCollectionPayloadDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nestedpayload", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested_payload_test.go"), []byte(nestedCollectionPayloadHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func nestedCollectionPayloadDSL() {
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	Service("batch", func() {
		unary := func(name, path string, payload any) {
			Method(name, func() {
				Payload(payload)
				Result(payload)
				HTTP(func() {
					POST(path)
				})
			})
		}
		stream := func(name, path string, payload any) {
			Method(name, func() {
				StreamingPayload(payload)
				StreamingResult(payload)
				HTTP(func() {
					GET(path)
				})
			})
		}
		unary("put", "/put", ArrayOf(item))
		unary("put nested", "/put-nested", ArrayOf(ArrayOf(item)))
		stream("talk", "/talk", ArrayOf(item))
		stream("talk nested", "/talk-nested", ArrayOf(ArrayOf(item)))
	})
}

const nestedCollectionPayloadHarness = `package nestedpayload

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

	"github.com/gorilla/websocket"

	batch "example.com/nestedpayload/gen/batch"
	client "example.com/nestedpayload/gen/http/batch/client"
	server "example.com/nestedpayload/gen/http/batch/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type bidi[T any] interface {
	RecvWithContext(context.Context) (T, error)
	SendWithContext(context.Context, T) error
}

func echo[T any](ctx context.Context, stream bidi[T], done chan error) error {
	err := func() error {
		for {
			msg, err := stream.RecvWithContext(ctx)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := stream.SendWithContext(ctx, msg); err != nil {
				return err
			}
		}
	}()
	done <- err
	return err
}

type service struct{ done chan error }

func (s *service) Put(_ context.Context, p []*batch.Item) ([]*batch.Item, error) {
	return p, nil
}

func (s *service) PutNested(_ context.Context, p [][]*batch.Item) ([][]*batch.Item, error) {
	return p, nil
}

func (s *service) Talk(ctx context.Context, stream batch.TalkServerStream) error {
	return echo[[]*batch.Item](ctx, stream, s.done)
}

func (s *service) TalkNested(ctx context.Context, stream batch.TalkNestedServerStream) error {
	return echo[[][]*batch.Item](ctx, stream, s.done)
}

func serve(t *testing.T) (*service, *httptest.Server) {
	t.Helper()
	s := &service{done: make(chan error, 1)}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(batch.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return s, hs
}

func newClient(hs *httptest.Server, scheme string) *client.Client {
	return client.NewClient(scheme, strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
}

func wait(t *testing.T, done chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("server did not finish the stream")
		return nil
	}
}

func TestUnaryRoundTrip(t *testing.T) {
	_, hs := serve(t)
	c := newClient(hs, "http")
	flat := []*batch.Item{{Name: "a"}, {Name: "b"}}
	if got, err := c.Put()(context.Background(), flat); err != nil || !reflect.DeepEqual(got, flat) {
		t.Errorf("put: got %+v, %v, want %+v", got, err, flat)
	}
	nested := [][]*batch.Item{{{Name: "a"}}, {{Name: "b"}, {Name: "c"}}}
	if got, err := c.PutNested()(context.Background(), nested); err != nil || !reflect.DeepEqual(got, nested) {
		t.Errorf("put nested: got %+v, %v, want %+v", got, err, nested)
	}
}

func TestStreamRoundTrip(t *testing.T) {
	s, hs := serve(t)
	c := newClient(hs, "ws")

	raw, err := c.Talk()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open talk: %v", err)
	}
	talk := raw.(*client.TalkClientStream)
	flat := []*batch.Item{{Name: "a"}, {Name: "b"}}
	if err := talk.Send(flat); err != nil {
		t.Fatalf("talk send: %v", err)
	}
	if got, err := talk.Recv(); err != nil || !reflect.DeepEqual(got, flat) {
		t.Errorf("talk: got %+v, %v, want %+v", got, err, flat)
	}
	if err := talk.Close(); err != nil {
		t.Fatalf("talk close: %v", err)
	}
	if err := wait(t, s.done); err != nil {
		t.Errorf("talk server: %v", err)
	}

	raw, err = c.TalkNested()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open talk nested: %v", err)
	}
	nestedStream := raw.(*client.TalkNestedClientStream)
	nested := [][]*batch.Item{{{Name: "a"}}, {{Name: "b"}, {Name: "c"}}}
	if err := nestedStream.Send(nested); err != nil {
		t.Fatalf("talk nested send: %v", err)
	}
	if got, err := nestedStream.Recv(); err != nil || !reflect.DeepEqual(got, nested) {
		t.Errorf("talk nested: got %+v, %v, want %+v", got, err, nested)
	}
	if err := nestedStream.Close(); err != nil {
		t.Fatalf("talk nested close: %v", err)
	}
	if err := wait(t, s.done); err != nil {
		t.Errorf("talk nested server: %v", err)
	}
}

func TestRejectsInvalidNestedElements(t *testing.T) {
	for _, message := range []string{` + "`" + `[[{}]]` + "`" + `, ` + "`" + `[[{"name":""}]]` + "`" + `, ` + "`" + `[[null]]` + "`" + `, ` + "`" + `[null]` + "`" + `} {
		t.Run("unary "+message, func(t *testing.T) {
			_, hs := serve(t)
			resp, err := http.Post(hs.URL+"/put-nested", "application/json", strings.NewReader(message))
			if err != nil {
				t.Fatalf("post: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status %d, want 400", resp.StatusCode)
			}
		})
		t.Run("stream "+message, func(t *testing.T) {
			s, hs := serve(t)
			conn, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(hs.URL, "http://")+"/talk-nested", nil)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				t.Fatalf("write: %v", err)
			}
			if err := wait(t, s.done); err == nil {
				t.Errorf("server accepted %s", message)
			}
		})
	}
}
`
