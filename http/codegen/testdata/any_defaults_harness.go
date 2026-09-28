package testdata

// AnyDefaultsHarness tests the raw JSON property presence contract. The caller
// substitutes TRANSPORT, RPC and SERVER_ARGS before compiling the module.
const AnyDefaultsHarness = `package anydefaults_test

import (
	"context"
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svc "example.com/anydefaults/gen/defaults"
	server "example.com/anydefaults/gen/TRANSPORT/defaults/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

func TestAnyDefaults(t *testing.T) {
	const rpc = RPC
	calls := 0
	var received any
	endpoint := func(_ context.Context, payload any) (any, error) {
		calls++
		received = reflect.ValueOf(payload).Elem().FieldByName("Metadata").Interface()
		return nil, nil
	}
	endpoints := new(svc.Endpoints)
	for index := range reflect.ValueOf(endpoints).Elem().NumField() {
		reflect.ValueOf(endpoints).Elem().Field(index).Set(reflect.ValueOf(loom.Endpoint(endpoint)))
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, SERVER_ARGS))
	for _, method := range []struct {
		name, fallback string
	}{
		{"text", "\"fallback\""},
		{"named", "\"fallback\""},
		{"boolean", "false"},
		{"number", "0"},
		{"empty", "\"\""},
		{"object", "{\"key\":\"value\"}"},
		{"array", "[\"value\"]"},
		{"nullable", "\"fallback\""},
		{"nodefault", "null"},
		{"required", ""},
	} {
		for _, input := range []struct {
			name, body, expected string
		}{
			{"absent", "{}", method.fallback},
			{"null", "{\"wire\":null}", "null"},
			{"text", "{\"wire\":\"explicit\"}", "\"explicit\""},
			{"empty", "{\"wire\":\"\"}", "\"\""},
			{"false", "{\"wire\":false}", "false"},
			{"zero", "{\"wire\":0}", "0"},
			{"object", "{\"wire\":{}}", "{}"},
			{"array", "{\"wire\":[]}", "[]"},
			{"malformed", "{x}", ""},
			{"truncated", "{", ""},
		} {
			t.Run(method.name+"/"+input.name, func(t *testing.T) {
				body, path := input.body, "/"+method.name
				if rpc {
					path = "/rpc"
					body = "{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"method\":\""+method.name+"\",\"params\":"+body+"}"
				}
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				before := calls
				mux.ServeHTTP(response, request)
				valid := input.expected != ""
				if !valid {
					require.Equal(t, before, calls, response.Body.String())
					if rpc {
						require.Equal(t, http.StatusOK, response.Code)
						var envelope struct {
							Error *struct {
								Code int
								Message string
								Data struct {
									Name string
								}
							}
						}
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope, json.MatchCaseInsensitiveNames(true)))
						require.NotNil(t, envelope.Error)
						if input.name == "absent" {
							require.Equal(t, -32602, envelope.Error.Code)
							require.Equal(t, "missing_field", envelope.Error.Data.Name)
							require.Equal(t, "Missing required field: metadata", envelope.Error.Message)
						} else {
							require.Equal(t, -32700, envelope.Error.Code)
							require.Equal(t, "Parse error", envelope.Error.Message)
						}
					} else {
						require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
						code := "decode_payload"
						if input.name == "absent" {
							code = "missing_field"
						}
						require.Contains(t, response.Body.String(), code)
					}
					return
				}
				status := http.StatusNoContent
				if rpc {
					status = http.StatusOK
					require.NotContains(t, response.Body.String(), "error")
				}
				require.Equal(t, status, response.Code, response.Body.String())
				require.Equal(t, before+1, calls)
				encoded, err := json.Marshal(received)
				require.NoError(t, err)
				require.JSONEq(t, input.expected, string(encoded))
				if method.name == "nodefault" && input.name == "absent" {
					require.True(t, reflect.ValueOf(received).IsNil())
				}
			})
		}
	}
}
`
