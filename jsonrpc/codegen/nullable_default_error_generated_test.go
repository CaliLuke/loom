package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestNullableDefaultJSONRPCGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.Service("rpcdefaults", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			for _, method := range []string{"single", "pruned", "watch", "unmapped"} {
				dsl.Method(method, func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
						dsl.Required("id")
					})
					if method == "watch" {
						dsl.StreamingResult(dsl.String)
					} else {
						dsl.Result(dsl.String)
					}
					dsl.Error("missing", func() {
						dsl.Nullable()
						dsl.Temporary()
					})
					dsl.JSONRPC(func() {
						if method == "watch" {
							dsl.ServerSentEvents()
						}
						if method == "unmapped" {
							return
						}
						dsl.Response("missing", -32004, func() {
							if method == "pruned" {
								dsl.Body(func() {
									dsl.Attribute("detail")
								})
							}
						})
					})
				})
			}
		})
		dsl.Service("socketdefaults", func() {
			dsl.JSONRPC(func() {
				dsl.GET("/socket")
			})
			dsl.Method("watch", func() {
				dsl.StreamingPayload(dsl.String)
				dsl.Result(dsl.String)
				dsl.Error("missing", func() {
					dsl.Nullable()
				})
				dsl.JSONRPC(func() {
					dsl.Response("missing", -32004)
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/defaultsrpc", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "default_test.go"), []byte(nullableDefaultRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const nullableDefaultRPCHarness = `package defaultsrpc_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	client "example.com/defaultsrpc/gen/jsonrpc/rpcdefaults/client"
	server "example.com/defaultsrpc/gen/jsonrpc/rpcdefaults/server"
	socketserver "example.com/defaultsrpc/gen/jsonrpc/socketdefaults/server"
	service "example.com/defaultsrpc/gen/rpcdefaults"
	sockets "example.com/defaultsrpc/gen/socketdefaults"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestNullableDefaultRPCWire(t *testing.T) {
	for _, null := range []bool{false, true} {
		failure := service.MakeMissing(errors.New("details"))
		if null {
			var err error
			failure, err = service.NewMissingError(loom.NullValue[loom.ServiceError]())
			require.NoError(t, err)
		}
		endpoint := func(context.Context, any) (any, error) {
			return nil, failure
		}
		endpoints := &service.Endpoints{Single: endpoint, Pruned: endpoint, Watch: endpoint, Unmapped: endpoint}
		mux := loomhttp.NewMuxer()
		server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
		rec := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"method\":\"single\",\"params\":{}}"))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, request)
		require.Equal(t, 200, rec.Code)
		var envelope jsonrpc.RawResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
		require.NotNil(t, envelope.Error)
		require.Equal(t, -32004, envelope.Error.Code)
		if null {
			require.Equal(t, "null", string(envelope.Error.Data))
		} else {
			require.Contains(t, string(envelope.Error.Data), "\"code\":\"missing\"")
			require.Contains(t, string(envelope.Error.Data), "\"temporary\":true")
		}
		fallback := httptest.NewRecorder()
		fallbackRequest := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":\"fallback\",\"method\":\"unmapped\",\"params\":{}}"))
		fallbackRequest.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(fallback, fallbackRequest)
		require.Equal(t, 200, fallback.Code)
		var fallbackEnvelope jsonrpc.RawResponse
		require.NoError(t, json.Unmarshal(fallback.Body.Bytes(), &fallbackEnvelope))
		require.NotNil(t, fallbackEnvelope.Error)
		require.Equal(t, -32603, fallbackEnvelope.Error.Code)
		var fallbackData jsonrpc.ErrorData
		require.NoError(t, json.Unmarshal(fallbackEnvelope.Error.Data, &fallbackData))
		require.Equal(t, "missing", fallbackData.Name)
		require.Equal(t, !null, fallbackData.Temporary)
		if null {
			require.Empty(t, fallbackData.ID)
		} else {
			require.NotEmpty(t, fallbackData.ID)
		}
		host := httptest.NewServer(mux)
		c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		_, err := c.Single()(context.Background(), &service.SinglePayload{ID: "1"})
		var decoded *service.MissingError
		require.ErrorAs(t, err, &decoded)
		payload, err := decoded.Payload()
		require.NoError(t, err)
		require.Equal(t, null, payload.IsNull())
		streamValue, err := c.Watch()(context.Background(), &service.WatchPayload{ID: "stream"})
		require.NoError(t, err)
		stream := streamValue.(service.WatchClientStream)
		_, err = stream.Recv()
		require.ErrorAs(t, err, &decoded)
		payload, err = decoded.Payload()
		require.NoError(t, err)
		require.Equal(t, null, payload.IsNull())
		require.NoError(t, stream.Close())
		if !null {
			_, err = c.Pruned()(context.Background(), &service.PrunedPayload{ID: "pruned"})
			require.ErrorAs(t, err, &decoded)
			payload, err = decoded.Payload()
			require.NoError(t, err)
			value, ok := payload.Value()
			require.True(t, ok)
			require.Equal(t, "missing", value.Name)
			require.Equal(t, "details", value.Message)
			require.False(t, value.Temporary)
		}
		host.Close()
	}
}

type socketImplementation struct {
	failure error
}

func (s socketImplementation) Watch(context.Context, string) (string, error) {
	return "", s.failure
}

func (s socketImplementation) HandleStream(ctx context.Context, stream sockets.Stream) error {
	if err := stream.SendError(ctx, "sent", s.failure); err != nil {
		return err
	}
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func TestNullableDefaultWebSocket(t *testing.T) {
	for _, null := range []bool{false, true} {
		failure := sockets.MakeMissing(errors.New("details"))
		if null {
			var err error
			failure, err = sockets.NewMissingError(loom.NullValue[loom.ServiceError]())
			require.NoError(t, err)
		}
		implementation := socketImplementation{failure}
		mux := loomhttp.NewMuxer()
		socketserver.Mount(mux, socketserver.New(implementation.HandleStream, sockets.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, loomhttp.WebSocketConfig{}))
		host := httptest.NewServer(mux)
		conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(host.URL, "http")+"/socket", nil)
		if response != nil && response.Body != nil {
			require.NoError(t, response.Body.Close())
		}
		require.NoError(t, err)
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
		for index := range 2 {
			if index == 1 {
				require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("{\"jsonrpc\":\"2.0\",\"id\":\"returned\",\"method\":\"watch\",\"params\":\"hello\"}")))
			}
			_, raw, err := conn.ReadMessage()
			require.NoError(t, err)
			var envelope jsonrpc.RawResponse
			require.NoError(t, json.Unmarshal(raw, &envelope))
			require.NotNil(t, envelope.Error)
			require.Equal(t, -32004, envelope.Error.Code)
			if null {
				require.Equal(t, "null", string(envelope.Error.Data))
			} else {
				require.Contains(t, string(envelope.Error.Data), "\"code\":\"missing\"")
			}
		}
		require.NoError(t, conn.Close())
		host.Close()
	}
}
`
