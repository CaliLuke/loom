package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNullableCollectionResponsesGenerated(t *testing.T) {
	cases := []struct {
		name                                 string
		shape                                func() expr.DataType
		empty, value, invalid, invalidDetail string
	}{
		{"array", func() expr.DataType {
			return ArrayOf(String)
		}, "[]", `["a","b"]`, `[null]`, "invalid null value"},
		{"map", func() expr.DataType {
			return MapOf(String, String)
		}, "{}", `{"a":"b"}`, `{"a":null}`, "invalid null value"},
		{"array nullable elements", func() expr.DataType {
			return ArrayOf(String, func() {
				Nullable()
			})
		}, "[]", `[null,"a"]`, `[1]`, "cannot unmarshal"},
		{"map nullable elements", func() expr.DataType {
			return MapOf(String, String, func() {
				Elem(func() {
					Nullable()
				})
			})
		}, "{}", `{"a":null,"b":"c"}`, `{"a":1}`, "cannot unmarshal"},
		{"nested array", func() expr.DataType {
			return ArrayOf(ArrayOf(String))
		}, "[]", `[["a"],[]]`, `[[null]]`, "invalid null value"},
		{"aliased map", func() expr.DataType {
			return Type("Inner", MapOf(String, String))
		}, "{}", `{"a":"b"}`, `{"a":null}`, "invalid null value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				values := Type("Values", c.shape(), func() {
					Nullable()
				})
				Service("collections", func() {
					Method("read", func() {
						Result(values)
						HTTP(func() {
							GET("/read")
						})
					})
					Method("watch", func() {
						StreamingResult(values)
						HTTP(func() {
							GET("/watch")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/collectionresponse", root)
			harness := strings.NewReplacer(
				"EMPTY_JSON", strconv.Quote(c.empty),
				"VALUE_JSON", strconv.Quote(c.value),
				"INVALID_JSON", strconv.Quote(c.invalid),
				"INVALID_DETAIL", strconv.Quote(c.invalidDetail),
			).Replace(nullableCollectionResponseHarness)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

const nullableCollectionResponseHarness = `package collectionresponse_test

import (
 "context"
 "encoding/json/v2"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"

 "example.com/collectionresponse/gen/collections"
 client "example.com/collectionresponse/gen/http/collections/client"
 server "example.com/collectionresponse/gen/http/collections/server"
 loomhttp "github.com/CaliLuke/loom/http"
 loom "github.com/CaliLuke/loom/pkg"
 "github.com/gorilla/websocket"
)

type service struct {
 value loom.Nullable[collections.Values]
}

func (s *service) Read(context.Context) (loom.Nullable[collections.Values], error) {
 return s.value, nil
}

func (s *service) Watch(ctx context.Context, stream collections.WatchServerStream) error {
 if err := stream.SendWithContext(ctx, s.value); err != nil {
  return err
 }
 return stream.Close()
}

func TestGeneratedResponseRoundTrips(t *testing.T) {
 for _, wire := range []string{"null", EMPTY_JSON, VALUE_JSON} {
  t.Run(wire, func(t *testing.T) {
   var value loom.Nullable[collections.Values]
   if err := json.Unmarshal([]byte(wire), &value); err != nil {
    t.Fatal(err)
   }
   mux := loomhttp.NewMuxer()
   server.Mount(mux, server.New(collections.NewEndpoints(&service{value}), mux,
    loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
   host := httptest.NewServer(mux)
   defer host.Close()
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   httpClient := newClient("http", host.URL)
   got, err := httpClient.Read()(ctx, nil)
   if err != nil {
    t.Fatal(err)
   }
   requireJSON(t, got, wire)
   wsClient := newClient("ws", host.URL)
   raw, err := wsClient.Watch()(ctx, nil)
   if err != nil {
    t.Fatal(err)
   }
   stream := raw.(collections.WatchClientStream)
   received, err := stream.RecvWithContext(ctx)
   if err != nil {
    t.Fatal(err)
   }
   requireJSON(t, received, wire)
   if _, err := stream.RecvWithContext(ctx); !errors.Is(err, io.EOF) {
    t.Errorf("end of response: %v", err)
   }
  })
 }
}

func TestGeneratedClientsRejectInvalidCollection(t *testing.T) {
 host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path == "/watch" {
   upgrader := websocket.Upgrader{}
   conn, err := upgrader.Upgrade(w, r, nil)
   if err != nil {
    t.Error(err)
    return
   }
   defer func() {
    if err := conn.Close(); err != nil {
     t.Error(err)
    }
   }()
   if err := conn.WriteMessage(websocket.TextMessage, []byte(INVALID_JSON)); err != nil {
    t.Error(err)
   }
   return
  }
  w.Header().Set("Content-Type", "application/json")
  if _, err := io.WriteString(w, INVALID_JSON); err != nil {
   t.Error(err)
  }
 }))
 defer host.Close()
 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 defer cancel()
 if _, err := newClient("http", host.URL).Read()(ctx, nil); err == nil || !strings.Contains(err.Error(), INVALID_DETAIL) {
  t.Errorf("HTTP client invalid collection: %v", err)
 }
 raw, err := newClient("ws", host.URL).Watch()(ctx, nil)
 if err != nil {
  t.Fatal(err)
 }
 if _, err := raw.(collections.WatchClientStream).RecvWithContext(ctx); err == nil || !strings.Contains(err.Error(), INVALID_DETAIL) {
  t.Errorf("WebSocket client invalid collection: %v", err)
 }
}

func newClient(scheme, address string) *client.Client {
 return client.NewClient(scheme, strings.TrimPrefix(address, "http://"), http.DefaultClient,
  loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
}

func requireJSON(t *testing.T, value any, want string) {
 t.Helper()
 got, err := json.Marshal(value, json.Deterministic(true))
 if err != nil || string(got) != want {
  t.Errorf("response=%s err=%v want=%s", got, err, want)
 }
}
`
