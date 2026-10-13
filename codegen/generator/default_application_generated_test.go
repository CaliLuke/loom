package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

const defaultApplicationHarness = `package presence_test

import (
	"context"
	json "encoding/json/v2"
	gc "example.com/presence/gen/grpc/proto/client"
	pb "example.com/presence/gen/grpc/proto/pb"
	gs "example.com/presence/gen/grpc/proto/server"
	pc "example.com/presence/gen/http/plain/client"
	ps "example.com/presence/gen/http/plain/server"
	rc "example.com/presence/gen/jsonrpc/rpc/client"
	rs "example.com/presence/gen/jsonrpc/rpc/server"
	plain "example.com/presence/gen/plain"
	psvc "example.com/presence/gen/proto"
	rpc "example.com/presence/gen/rpc"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *plain.State
		input string
	}{
		{"explicit zero", &plain.State{Flag: false, Count: 0}, ""},
		{"CLI absent defaults", &plain.State{Flag: true, Count: 7, Label: "fallback"}, "{}"},
		{"CLI populated", &plain.State{Flag: true, Count: 9, Label: "value"}, "{\"flag\":true,\"count\":9,\"label\":\"value\"}"},
		{"CLI supplied zeros", &plain.State{}, "{\"flag\":false,\"count\":0,\"label\":\"\"}"},
		{"nonzero control", &plain.State{Flag: true, Count: 9, Label: "value", Text: ptr("value"), Bytes: []byte("value"), List: []string{"value"}, Mapping: map[string]string{"key": "value"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received *plain.State
			endpoints := &plain.Endpoints{Echo: func(_ context.Context, v any) (any, error) {
				calls++
				received = v.(*plain.State)
				return v, nil
			}}
			mux := loomhttp.NewMuxer()
			ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			host := httptest.NewServer(mux)
			defer host.Close()
			client := pc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			payload := tc.value
			if tc.input != "" {
				var err error
				payload, err = pc.BuildEchoPayload(tc.input)
				require.NoError(t, err)
				require.Equal(t, tc.value, payload)
			}
			got, err := client.Echo()(context.Background(), payload)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			t.Run("request", func(t *testing.T) {
				require.Equal(t, tc.value, received)
			})
			t.Run("response", func(t *testing.T) {
				require.Equal(t, tc.value, got)
			})
		})
	}
}
func TestJSONRPC(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *rpc.State
		input string
	}{
		{"explicit zero", &rpc.State{Flag: false, Count: 0}, ""},
		{"CLI absent defaults", &rpc.State{Flag: true, Count: 7, Label: "fallback"}, "{}"},
		{"CLI populated", &rpc.State{Flag: true, Count: 9, Label: "value"}, "{\"flag\":true,\"count\":9,\"label\":\"value\"}"},
		{"CLI supplied zeros", &rpc.State{}, "{\"flag\":false,\"count\":0,\"label\":\"\"}"},
		{"nonzero control", &rpc.State{Flag: true, Count: 9, Label: "value", Text: ptr("value"), Bytes: []byte("value"), List: []string{"value"}, Mapping: map[string]string{"key": "value"}}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received *rpc.State
			endpoints := &rpc.Endpoints{Echo: func(_ context.Context, v any) (any, error) {
				calls++
				received = v.(*rpc.State)
				return v, nil
			}}
			mux := loomhttp.NewMuxer()
			rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			host := httptest.NewServer(mux)
			defer host.Close()
			client := rc.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			payload := tc.value
			if tc.input != "" {
				var err error
				payload, err = rc.BuildEchoPayload(tc.input)
				require.NoError(t, err)
				require.Equal(t, tc.value, payload)
			}
			got, err := client.Echo()(context.Background(), payload)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			t.Run("request", func(t *testing.T) {
				require.Equal(t, tc.value, received)
			})
			t.Run("response", func(t *testing.T) {
				require.Equal(t, tc.value, got)
			})
		})
	}
}
func TestGRPC(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value *psvc.State
	}{
		{"explicit zero", &psvc.State{Flag: false, Count: 0}},
		{"nonzero control", &psvc.State{Flag: true, Count: 9, Label: "value", Text: ptr("value"), Bytes: []byte("value"), List: []string{"value"}, Mapping: map[string]string{"key": "value"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := proto.Marshal(gc.NewProtoEchoRequest(tc.value))
			require.NoError(t, err)
			decoded := new(pb.EchoRequest)
			require.NoError(t, proto.Unmarshal(wire, decoded))
			got, err := gs.DecodeEchoRequest(context.Background(), decoded, nil)
			require.NoError(t, err)
			require.Equal(t, tc.value, got)
		})
	}
}

func TestRawHTTPPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       *plain.State
	}{
		{"absent defaults", "{}", &plain.State{Flag: true, Count: 7, Label: "fallback"}},
		{"explicit zero", "{\"flag\":false,\"count\":0,\"label\":\"\"}", &plain.State{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received *plain.State
			endpoints := &plain.Endpoints{Echo: func(_ context.Context, v any) (any, error) {
				calls++
				received = v.(*plain.State)
				return tc.want, nil
			}}
			mux := loomhttp.NewMuxer()
			ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			body, path := tc.body, "/echo"
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, 1, calls)
			t.Run("decoded request", func(t *testing.T) {
				require.Equal(t, tc.want, received)
			})
			var wire map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &wire))
			t.Run("encoded response", func(t *testing.T) {
				require.Equal(t, tc.want.Flag, wire["flag"])
				require.Equal(t, float64(tc.want.Count), wire["count"])
				require.Equal(t, string(tc.want.Label), wire["label"])
			})
		})
	}
}

func TestRawJSONRPCPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       *rpc.State
	}{
		{"absent defaults", "{}", &rpc.State{Flag: true, Count: 7, Label: "fallback"}},
		{"explicit zero", "{\"flag\":false,\"count\":0,\"label\":\"\"}", &rpc.State{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received *rpc.State
			endpoints := &rpc.Endpoints{Echo: func(_ context.Context, v any) (any, error) {
				calls++
				received = v.(*rpc.State)
				return tc.want, nil
			}}
			mux := loomhttp.NewMuxer()
			rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			body, path := tc.body, "/echo"
			path = "/rpc"
			body = "{\"jsonrpc\":\"2.0\",\"id\":\"test\",\"method\":\"echo\",\"params\":" + body + "}"
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, 1, calls)
			t.Run("decoded request", func(t *testing.T) {
				require.Equal(t, tc.want, received)
			})
			var wire map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &wire))
			require.NotContains(t, wire, "error")
			wire = wire["result"].(map[string]any)
			t.Run("encoded response", func(t *testing.T) {
				require.Equal(t, tc.want.Flag, wire["flag"])
				require.Equal(t, float64(tc.want.Count), wire["count"])
				require.Equal(t, string(tc.want.Label), wire["label"])
			})
		})
	}
}

func TestRequiredNullableHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, want string
		invalid                  bool
	}{
		{"required absent", "required", "{}", "", true},
		{"required zero", "required", "{\"value\":false}", "{\"value\":false}", false},
		{"nullable absent", "nullable", "{}", "{\"value\":\"fallback\"}", false},
		{"nullable null", "nullable", "{\"value\":null}", "{\"value\":null}", false},
		{"nullable empty", "nullable", "{\"value\":\"\"}", "{\"value\":\"\"}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received any
			endpoints := &plain.Endpoints{
				Required: func(_ context.Context, v any) (any, error) {
					calls++
					received = v
					return nil, nil
				},
				Nullable: func(_ context.Context, v any) (any, error) {
					calls++
					received = v
					return v, nil
				},
			}
			mux := loomhttp.NewMuxer()
			ps.Mount(mux, ps.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			body, path := tc.body, "/"+tc.method
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if tc.invalid {
				require.Zero(t, calls, response.Body.String())
				require.Equal(t, http.StatusBadRequest, response.Code)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, "missing_field", problem.Code)
				require.Equal(t, "Missing required field: value", problem.Detail)
				return
			}
			require.Equal(t, 1, calls, response.Body.String())
			status := http.StatusOK
			if tc.method == "required" {
				status = http.StatusNoContent
			}
			require.Equal(t, status, response.Code)
			wire, err := json.Marshal(received)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}

func TestRequiredNullableJSONRPC(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, want string
		invalid                  bool
	}{
		{"required absent", "required", "{}", "", true},
		{"required zero", "required", "{\"value\":false}", "{\"value\":false}", false},
		{"nullable absent", "nullable", "{}", "{\"value\":\"fallback\"}", false},
		{"nullable null", "nullable", "{\"value\":null}", "{\"value\":null}", false},
		{"nullable empty", "nullable", "{\"value\":\"\"}", "{\"value\":\"\"}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received any
			endpoints := &rpc.Endpoints{
				Required: func(_ context.Context, v any) (any, error) {
					calls++
					received = v
					return nil, nil
				},
				Nullable: func(_ context.Context, v any) (any, error) {
					calls++
					received = v
					return v, nil
				},
			}
			mux := loomhttp.NewMuxer()
			rs.Mount(mux, rs.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			body, path := tc.body, "/"+tc.method
			path = "/rpc"
			body = "{\"jsonrpc\":\"2.0\",\"id\":\"test\",\"method\":\"" + tc.method + "\",\"params\":" + body + "}"
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if tc.invalid {
				require.Zero(t, calls, response.Body.String())
				require.Equal(t, http.StatusOK, response.Code)
				var result map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				failure, ok := result["error"].(map[string]any)
				require.True(t, ok, response.Body.String())
				require.Equal(t, float64(-32602), failure["code"])
				require.Equal(t, "Missing required field: value", failure["message"])
				return
			}
			require.Equal(t, 1, calls, response.Body.String())
			require.Equal(t, http.StatusOK, response.Code)
			wire, err := json.Marshal(received)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(wire))
		})
	}
}
func ptr[T any](v T) *T {
	return &v
}
`

func TestDefaultApplicationGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/presence", defaultApplicationDSL, defaultApplicationHarness)
}

func defaultApplicationDSL() {
	dsl.API("presence", func() {
		dsl.JSONRPC(func() {
		})
	})
	label := dsl.Type("Label", dsl.String)
	state := dsl.Type("State", func() {
		dsl.Field(1, "flag", dsl.Boolean, func() {
			dsl.Default(true)
		})
		dsl.Field(2, "count", dsl.Int, func() {
			dsl.Default(7)
		})
		dsl.Field(3, "text", dsl.String)
		dsl.Field(4, "bytes", dsl.Bytes)
		dsl.Field(5, "list", dsl.ArrayOf(dsl.String))
		dsl.Field(6, "mapping", dsl.MapOf(dsl.String, dsl.String))
		dsl.Field(7, "label", label, func() {
			dsl.Default("fallback")
		})
	})
	nullable := dsl.Type("NullableState", func() {
		dsl.Attribute("value", dsl.String, func() {
			dsl.Nullable()
			dsl.Default("fallback")
		})
	})
	for _, name := range []string{"plain", "rpc", "proto"} {
		dsl.Service(name, func() {
			if name == "rpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			if name != "proto" {
				dsl.Method("required", func() {
					dsl.Payload(func() {
						dsl.Attribute("value", dsl.Boolean, func() {
							dsl.Default(true)
						})
						dsl.Required("value")
					})
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/required")
						})
					}
				})
				dsl.Method("nullable", func() {
					dsl.Payload(nullable)
					dsl.Result(nullable)
					if name == "rpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.POST("/nullable")
						})
					}
				})
			}
			dsl.Method("echo", func() {
				dsl.Payload(state)
				dsl.Result(state)
				switch name {
				case "plain":
					dsl.HTTP(func() {
						dsl.POST("/echo")
					})
				case "rpc":
					dsl.JSONRPC(func() {
					})
				case "proto":
					dsl.GRPC(func() {
					})
				}
			})
		})
	}
}
