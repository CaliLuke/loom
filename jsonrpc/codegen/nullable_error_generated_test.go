package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCNullableErrorGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, nullableJSONRPCErrorDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/nullablerpc", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_test.go"), []byte(nullableJSONRPCErrorHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func nullableJSONRPCErrorDSL() {
	failure := dsl.Type("Failure", func() {
		dsl.ErrorName("name", dsl.String)
		dsl.Attribute("message", dsl.String)
		dsl.Required("name", "message")
	})
	reply := dsl.Type("Reply", func() {
		dsl.Attribute("message", dsl.String)
		dsl.Required("message")
	})
	dsl.Service("nullable", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		for _, method := range []string{"single", "ambiguous", "watch", "selected_result", "selected_error"} {
			dsl.Method(method, func() {
				if method != "selected_result" && method != "selected_error" {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
						dsl.Required("id")
					})
				}

				switch method {
				case "watch":
					dsl.StreamingResult(dsl.String)
				case "selected_result":
					dsl.Result(reply, func() {
						dsl.Nullable()
					})
				default:
					dsl.Result(dsl.String)
				}
				dsl.Error("first", failure, func() {
					dsl.Nullable()
				})
				if method == "ambiguous" {
					dsl.Error("second", failure, func() {
						dsl.Nullable()
					})
				}
				dsl.JSONRPC(func() {
					if method == "watch" {
						dsl.ServerSentEvents()
					}
					if method == "selected_result" {
						dsl.Response(dsl.StatusOK, func() {
							dsl.Body("message")
						})
					}
					dsl.Response("first", 4001, func() {
						if method == "selected_error" {
							dsl.Body(func() {
								dsl.Attribute("name", dsl.String)
								dsl.Attribute("message", dsl.String)
								dsl.Required("name", "message")
							})
						}
					})
					if method == "ambiguous" {
						dsl.Response("second", 4001)
					}
				})
			})
		}
	})
}

const nullableJSONRPCErrorHarness = `package nullablerpc_test

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/jsonrpc"

	client "example.com/nullablerpc/gen/jsonrpc/nullable/client"
	server "example.com/nullablerpc/gen/jsonrpc/nullable/server"
	service "example.com/nullablerpc/gen/nullable"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestNullableRPCWire(t *testing.T) {
	for _, null := range []bool{false, true} {
		payload := loom.NullableValue(service.Failure{Name: "first", Message: "details"})
		if null {
			payload = loom.NullValue[service.Failure]()
		}
		failure, err := service.NewFirstError(payload)
		require.NoError(t, err)
		endpoint := func(context.Context, any) (any, error) { return nil, failure }
		endpoints := &service.Endpoints{Single: endpoint, Ambiguous: endpoint, Watch: endpoint}
		mux := loomhttp.NewMuxer()
		generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, err error) { t.Errorf("server error: %v", err) })
		server.Mount(mux, generated)
		rec := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(` + "`" + `{"jsonrpc":"2.0","id":"1","method":"single","params":{}}` + "`" + `))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, request)
		require.Equal(t, 200, rec.Code)
		var envelope jsonrpc.RawResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
		require.NotNil(t, envelope.Error)
		require.Equal(t, 4001, envelope.Error.Code)
		if null {
			require.Equal(t, "null", string(envelope.Error.Data))
		} else {
			require.JSONEq(t, ` + "`" + `{"name":"first","message":"details"}` + "`" + `, string(envelope.Error.Data))
		}
		transport := httptest.NewServer(mux)
		defer transport.Close()
		c := client.NewClient("http", strings.TrimPrefix(transport.URL, "http://"), transport.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		_, err = c.Single()(context.Background(), &service.SinglePayload{ID: "1"})
		var decoded *service.FirstError
		require.ErrorAs(t, err, &decoded)
		value, err := decoded.Payload()
		require.NoError(t, err)
		require.True(t, value.Present())
		require.Equal(t, null, value.IsNull())
		streamValue, err := c.Watch()(context.Background(), &service.WatchPayload{ID: "stream"})
		require.NoError(t, err)
		stream := streamValue.(service.WatchClientStream)
		_, err = stream.Recv()
		require.ErrorAs(t, err, &decoded)
		streamPayload, payloadErr := decoded.Payload()
		require.NoError(t, payloadErr)
		require.Equal(t, null, streamPayload.IsNull())
		require.NoError(t, stream.Close())
		_, err = c.Ambiguous()(context.Background(), &service.AmbiguousPayload{ID: "2"})
		if null {
			var invalid *loomhttp.ClientError
			require.ErrorAs(t, err, &invalid)
			require.Contains(t, invalid.Message, "ambiguous")
		} else {
			require.ErrorAs(t, err, &decoded)
		}
	}
}

func TestNullablePreparationFailures(t *testing.T) {
	for _, null := range []bool{false, true} {
		for _, method := range []string{"selected_result", "selected_error"} {
			t.Run(fmt.Sprintf("%s/null=%t", method, null), func(t *testing.T) {
				result := loom.NullableValue(service.Reply{Message: "details"})
				payload := loom.NullableValue(service.Failure{Name: "first", Message: "details"})
				if null {
					result = loom.NullValue[service.Reply]()
					payload = loom.NullValue[service.Failure]()
				}
				failure, err := service.NewFirstError(payload)
				require.NoError(t, err)
				calls, reports := 0, 0
				endpoints := &service.Endpoints{
					SelectedResult: func(context.Context, any) (any, error) {
						calls++
						return result, nil
					},
					SelectedError: func(context.Context, any) (any, error) {
						calls++
						return nil, failure
					},
				}
				mux := loomhttp.NewMuxer()
				generated := server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, func(_ context.Context, _ http.ResponseWriter, _ error) {
					reports++
				})
				server.Mount(mux, generated)
				for _, hasID := range []bool{true, false} {
					id := ""
					if hasID {
						id = ` + "`" + `,"id":"prepared"` + "`" + `
					}
					before, priorReports := calls, reports
					request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(` + "`" + `{"jsonrpc":"2.0","method":"` + "`" + `+method+` + "`" + `","params":{}` + "`" + `+id+"}"))
					request.Header.Set("Content-Type", "application/json")
					rec := httptest.NewRecorder()
					mux.ServeHTTP(rec, request)
					require.Equal(t, before+1, calls)
					if !hasID {
						require.Empty(t, rec.Body.String())
						if method == "selected_error" {
							require.Equal(t, priorReports+1, reports)
						} else {
							require.Equal(t, priorReports, reports)
						}
						continue
					}
					require.Equal(t, priorReports, reports, "preparation must complete a protocol error, not report a transport failure")
					require.Equal(t, http.StatusOK, rec.Code)
					var envelope jsonrpc.RawResponse
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope), rec.Body.String())
					require.Equal(t, "prepared", envelope.ID)
					if null {
						require.NotNil(t, envelope.Error)
						require.Equal(t, -32603, envelope.Error.Code)
						require.Equal(t, "internal server error", envelope.Error.Message)
						require.Empty(t, envelope.Result)
					} else if method == "selected_error" {
						require.NotNil(t, envelope.Error)
						require.Equal(t, 4001, envelope.Error.Code)
						require.JSONEq(t, ` + "`" + `{"name":"first","message":"details"}` + "`" + `, string(envelope.Error.Data))
					} else {
						require.Nil(t, envelope.Error)
						require.JSONEq(t, ` + "`" + `"details"` + "`" + `, string(envelope.Result))
					}
				}
			})
		}
	}
}
`
