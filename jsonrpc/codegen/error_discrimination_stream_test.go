package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

const jsonRPCStreamErrorDiscriminationHarness = `package streamerrors_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/streamerrors/gen/events"
	eventsclient "example.com/streamerrors/gen/jsonrpc/events/client"
	eventsserver "example.com/streamerrors/gen/jsonrpc/events/server"
	socketclient "example.com/streamerrors/gen/jsonrpc/socket/client"
	socketserver "example.com/streamerrors/gen/jsonrpc/socket/server"
	socket "example.com/streamerrors/gen/socket"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
	"github.com/gorilla/websocket"
)

type (
	eventService  struct{ send bool }
	socketService struct{ send bool }
)

func TestTypedStreamErrors(t *testing.T) {
	for _, name := range []string{"forbidden", "denied"} {
		t.Run("sse/"+name, func(t *testing.T) {
			host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, err := fmt.Fprintf(w, "event: response\ndata: {\"jsonrpc\":\"2.0\",\"id\":\"id\",\"error\":{\"code\":4403,\"message\":\"safe\",\"data\":{\"error_kind\":%q,\"name\":\"unrelated resource\"}}}\n\n", name)
				if err != nil {
					t.Error(err)
				}
			}))
			defer host.Close()
			client := eventsclient.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			value, err := client.Watch()(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			stream := value.(events.WatchClientStream)
			defer func() {
				if err := stream.Close(); err != nil {
					t.Error(err)
				}
			}()
			_, err = stream.Recv()
			var typed *events.Failure
			if !errors.As(err, &typed) {
				t.Errorf("typed %s: got %T %v", name, err, err)
			} else if typed.Kind != name || typed.Name != "unrelated resource" {
				t.Errorf("wrong identity: %+v", typed)
			}
		})
		t.Run("ws/"+name, func(t *testing.T) {
			host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				_, request, err := conn.ReadMessage()
				if err != nil {
					t.Error(err)
					return
				}
				var envelope jsonrpc.RawRequest
				if err := json.Unmarshal(request, &envelope); err != nil {
					t.Error(err)
					return
				}
				id, err := json.Marshal(envelope.ID)
				if err != nil {
					t.Error(err)
					return
				}
				response := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%s,\"error\":{\"code\":4403,\"message\":\"safe\",\"data\":{\"error_kind\":%q,\"name\":\"unrelated resource\"}}}", id, name)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
					t.Error(err)
				}
			}))
			defer host.Close()
			client := socketclient.NewClient("ws", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, loomhttp.WebSocketConfig{})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			value, err := client.Watch()(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			stream := value.(socket.WatchClientStream)
			defer func() {
				if err := stream.Close(); err != nil {
					t.Error(err)
				}
			}()
			_, err = stream.Recv()
			var typed *socket.Failure
			if !errors.As(err, &typed) {
				t.Errorf("typed %s: got %T %v", name, err, err)
			} else if typed.Kind != name || typed.Name != "unrelated resource" {
				t.Errorf("wrong identity: %+v", typed)
			}
		})
	}
}
func TestMountedTypedStreamErrors(t *testing.T) {
	for _, send := range []bool{false, true} {
		t.Run(fmt.Sprintf("send=%t", send), func(t *testing.T) {
			mux := loomhttp.NewMuxer()
			eventsserver.Mount(mux, eventsserver.New(events.NewEndpoints(eventService{send: send}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			ws := socketService{send: send}
			socketserver.Mount(mux, socketserver.New(ws.HandleStream, socket.NewEndpoints(ws), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, loomhttp.WebSocketConfig{}))
			host := httptest.NewServer(mux)
			defer host.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ec := eventsclient.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			wc := socketclient.NewClient("ws", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, loomhttp.WebSocketConfig{})
			for _, tc := range []struct {
				name string
				open func(context.Context, any) (any, error)
			}{
				{"sse", ec.Watch()}, {"ws call", wc.Watch()}, {"ws receive", wc.Chat()},
			} {
				t.Run(tc.name, func(t *testing.T) {
					value, err := tc.open(ctx, nil)
					if err != nil {
						t.Fatal(err)
					}
					stream := value.(interface {
						Recv() (string, error)
						Close() error
					})
					defer func() {
						if err := stream.Close(); err != nil {
							t.Error(err)
						}
					}()
					if chat, ok := value.(socket.ChatClientStream); ok {
						if err := chat.Send("request"); err != nil {
							t.Fatal(err)
						}
					}
					_, err = stream.Recv()
					var identity interface{ LoomErrorName() string }
					if !errors.As(err, &identity) || identity.LoomErrorName() != "forbidden" {
						t.Errorf("expected mapped forbidden, got %T %v", err, err)
					}
				})
			}
		})
	}
}

func (s eventService) Watch(ctx context.Context, stream events.WatchServerStream) error {
	failure := &events.Failure{Kind: "forbidden", Name: "unrelated resource"}
	if s.send {
		return stream.SendError(ctx, "id", failure)
	}
	return failure
}

func (s socketService) HandleStream(ctx context.Context, stream socket.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (s socketService) Watch(ctx context.Context, stream socket.WatchServerStream) error {
	failure := &socket.Failure{Kind: "forbidden", Name: "unrelated resource"}
	if s.send {
		return stream.SendError(ctx, failure)
	}
	return failure
}

func (s socketService) Chat(_ context.Context, _ string, _ socket.ChatServerStream) error {
	return &socket.Failure{Kind: "forbidden", Name: "unrelated resource"}
}
`

// TestJSONRPCStreamErrorDiscriminator checks generated SSE and WebSocket clients.
func TestJSONRPCStreamErrorDiscriminator(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCStreamErrorDiscriminationDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_ERROR_DISCRIMINATION_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o700))
	}
	renderJSONRPCModule(t, dir, "example.com/streamerrors", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error_test.go"), []byte(jsonRPCStreamErrorDiscriminationHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-count=1", "./...")
}

func jsonRPCStreamErrorDiscriminationDSL() {
	failure := dsl.Type("Failure", func() {
		dsl.ErrorName("kind:error_kind", dsl.String)
		dsl.Attribute("name", dsl.String)
		dsl.Required("kind", "name")
	})
	for _, transport := range []string{"events", "socket"} {
		dsl.Service(transport, func() {
			dsl.JSONRPC(func() {
				if transport == "events" {
					dsl.POST("/events")
				} else {
					dsl.GET("/socket")
				}
			})
			dsl.Method("watch", func() {
				dsl.StreamingResult(dsl.String)
				dsl.Error("forbidden", failure)
				dsl.Error("denied", failure)
				dsl.JSONRPC(func() {
					if transport == "events" {
						dsl.ServerSentEvents()
					}
					dsl.Response("forbidden", 4403)
					dsl.Response("denied", 4403)
				})
			})
			if transport == "socket" {
				dsl.Method("chat", func() {
					dsl.StreamingPayload(dsl.String)
					dsl.StreamingResult(dsl.String)
					dsl.Error("forbidden", failure)
					dsl.Error("denied", failure)
					dsl.JSONRPC(func() {
						dsl.Response("forbidden", 4403)
						dsl.Response("denied", 4403)
					})
				})
			}
		})
	}
}
