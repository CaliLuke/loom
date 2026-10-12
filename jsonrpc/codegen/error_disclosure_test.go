package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestGeneratedErrorDisclosure(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcErrorDisclosureDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/disclosure", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "disclosure_test.go"), []byte(jsonrpcErrorDisclosureHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcErrorDisclosureDSL() {
	dsl.API("disclosure", func() {
		dsl.JSONRPC(func() {})
	})
	for _, name := range []string{"unary", "events", "socket"} {
		dsl.Service(name, func() {
			dsl.JSONRPC(func() {
				if name == "socket" {
					dsl.GET("/" + name)
				} else {
					dsl.POST("/" + name)
				}
			})
			dsl.Method("show", func() {
				dsl.Payload(dsl.String)
				if name == "unary" {
					dsl.Result(dsl.String)
				} else {
					dsl.StreamingResult(dsl.String)
				}
				dsl.JSONRPC(func() {
					if name == "events" {
						dsl.ServerSentEvents()
					}
				})
			})
		})
	}
}

const jsonrpcErrorDisclosureHarness = `package disclosure_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/disclosure/gen/events"
	eventserver "example.com/disclosure/gen/jsonrpc/events/server"
	socketserver "example.com/disclosure/gen/jsonrpc/socket/server"
	unaryserver "example.com/disclosure/gen/jsonrpc/unary/server"
	socket "example.com/disclosure/gen/socket"
	unary "example.com/disclosure/gen/unary"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestPublicMessages(t *testing.T) {
	secret := errors.New("private database password")
	public := loom.NewServiceError(secret, "public", false, false, true)
	public.Message = "public detail"
	safe := loom.WithErrorRemedy(loom.NewServiceError(secret, "public", false, false, true), &loom.ErrorRemedy{SafeMessage: "safe detail"})
	for _, tc := range []struct {
		name    string
		err     error
		message string
	}{
		{"plain", secret, "internal server error"},
		{"wrapped", fmt.Errorf("private query: %w", secret), "internal server error"},
		{"public wrapper", fmt.Errorf("private query: %w", public), "public detail"},
		{"safe", safe, "safe detail"},
		{"join", errors.Join(public, secret), "internal server error"},
	} {
		for _, transport := range []string{"unary", "events", "socket"} {
			for _, send := range []bool{false, true} {
				if transport == "unary" && send {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/send=%t", tc.name, transport, send), func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
					defer cancel()
					before := tc.err.Error()
					called := make(chan error, 1)
					endpoint := func(ctx context.Context, input any) (any, error) {
						called <- tc.err
						if send {
							switch input := input.(type) {
							case *events.ShowEndpointInput:
								return nil, input.Stream.SendError(ctx, "request", tc.err)
							case *socket.ShowEndpointInput:
								return nil, input.Stream.SendError(ctx, tc.err)
							default:
								return nil, fmt.Errorf("unexpected endpoint input %T", input)
							}
						}
						return nil, tc.err
					}
					mux := loomhttp.NewMuxer()
					unaryserver.Mount(mux, unaryserver.New(&unary.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
					eventserver.Mount(mux, eventserver.New(&events.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
					socketserver.Mount(mux, socketserver.New(func(ctx context.Context, stream socket.Stream) error {
						for {
							if err := stream.Recv(ctx); err != nil {
								return err
							}
						}
					}, &socket.Endpoints{Show: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
					host := httptest.NewServer(mux)
					defer host.Close()
					request := "{\"jsonrpc\":\"2.0\",\"method\":\"show\",\"params\":\"value\",\"id\":\"request\"}"
					var wire []byte
					if transport == "socket" {
						conn, response, err := websocket.DefaultDialer.DialContext(ctx, "ws"+strings.TrimPrefix(host.URL, "http")+"/socket", nil)
						require.NoError(t, err)
						require.NoError(t, response.Body.Close())
						defer func() {
							require.NoError(t, conn.Close())
						}()
						require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
						require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(request)))
						_, wire, err = conn.ReadMessage()
						require.NoError(t, err)
					} else {
						req, err := http.NewRequestWithContext(ctx, http.MethodPost, host.URL+"/"+transport, strings.NewReader(request))
						require.NoError(t, err)
						req.Header.Set("Content-Type", "application/json")
						if transport == "events" {
							req.Header.Set("Accept", "text/event-stream")
						}
						response, err := host.Client().Do(req)
						require.NoError(t, err)
						wire, err = io.ReadAll(response.Body)
						require.NoError(t, err)
						require.NoError(t, response.Body.Close())
						require.Equal(t, http.StatusOK, response.StatusCode)
					}
					require.NotContains(t, string(wire), "private")
					if transport == "events" {
						for _, line := range strings.Split(string(wire), "\n") {
							if strings.HasPrefix(line, "data:") {
								wire = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
								break
							}
						}
					}
					var envelope map[string]any
					require.NoError(t, json.Unmarshal(wire, &envelope))
					require.Equal(t, "request", envelope["id"])
					problem, ok := envelope["error"].(map[string]any)
					require.True(t, ok)
					require.Equal(t, float64(-32603), problem["code"])
					require.Equal(t, tc.message, problem["message"])
					select {
					case original := <-called:
						require.Same(t, tc.err, original)
						require.Equal(t, before, original.Error())
						require.ErrorIs(t, original, secret)
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				})
			}
		}
	}
}
`
