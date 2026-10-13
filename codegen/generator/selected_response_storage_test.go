package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestSelectedResponseStorageGeneratedMixed(t *testing.T) {
	dir := t.TempDir()
	renderDesignModule(t, dir, "example.com/responsestorage", httptestdata.SelectedResponseStorageDSL)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "storage_test.go"), []byte(selectedResponseStorageHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

const selectedResponseStorageHarness = `package storage_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"encoding/json/jsontext"
	client "example.com/responsestorage/gen/http/httpfields/client"
	server "example.com/responsestorage/gen/http/httpfields/server"
	svc "example.com/responsestorage/gen/httpfields"
	views "example.com/responsestorage/gen/httpfields/views"
	rpcclient "example.com/responsestorage/gen/jsonrpc/jsonrpcfields/client"
	rpcserver "example.com/responsestorage/gen/jsonrpc/jsonrpcfields/server"
	rpcsvc "example.com/responsestorage/gen/jsonrpcfields"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type storageCase struct {
	name                                     string
	httpConstructor, rpcConstructor, project any
	values                                   []string
}

func cases() []storageCase {
	return []storageCase{
		{"required", client.NewRequiredHttprequiredOK, rpcclient.NewRequiredJsonrpcrequiredOK, svc.NewViewedHttprequired, []string{"\"hello\"", "\"\""}},
		{"optional", client.NewOptionalHttpoptionalOK, rpcclient.NewOptionalJsonrpcoptionalOK, svc.NewViewedHttpoptional, []string{"\"hello\"", "\"\""}},
		{"defaulted", client.NewDefaultedHttpdefaultedOK, rpcclient.NewDefaultedJsonrpcdefaultedOK, svc.NewViewedHttpdefaulted, []string{"\"hello\"", "\"\""}},
		{"named", client.NewNamedHttpnamedOK, rpcclient.NewNamedJsonrpcnamedOK, svc.NewViewedHttpnamed, []string{"\"hello\"", "\"\""}},
		{"namedoptional", client.NewNamedoptionalHttpnamedoptionalOK, rpcclient.NewNamedoptionalJsonrpcnamedoptionalOK, svc.NewViewedHttpnamedoptional, []string{"\"hello\"", "\"\""}},
		{"nullable", client.NewNullableHttpnullableOK, rpcclient.NewNullableJsonrpcnullableOK, svc.NewViewedHttpnullable, []string{"\"hello\"", "\"\"", "null"}},
		{"nullableoptional", client.NewNullableoptionalHttpnullableoptionalOK, rpcclient.NewNullableoptionalJsonrpcnullableoptionalOK, svc.NewViewedHttpnullableoptional, []string{"\"hello\"", "null"}},
		{"number", client.NewNumberHttpnumberOK, rpcclient.NewNumberJsonrpcnumberOK, svc.NewViewedHttpnumber, []string{"23", "0"}},
		{"custom", client.NewCustomHttpcustomOK, rpcclient.NewCustomJsonrpccustomOK, svc.NewViewedHttpcustom, []string{"\"2026-01-02T03:04:05Z\""}},
		{"array", client.NewArrayHttparrayOK, rpcclient.NewArrayJsonrpcarrayOK, svc.NewViewedHttparray, []string{"[\"hello\"]", "[]"}},
		{"bytes", client.NewBytesHttpbytesOK, rpcclient.NewBytesJsonrpcbytesOK, svc.NewViewedHttpbytes, []string{"\"aGk=\"", "\"\""}},
		{"ordinary", client.NewOrdinaryHttpordinaryOK, rpcclient.NewOrdinaryJsonrpcordinaryOK, nil, []string{"\"hello\"", "\"\""}},
		{"ordinaryoptional", client.NewOrdinaryoptionalHttpordinaryoptionalOK, rpcclient.NewOrdinaryoptionalJsonrpcordinaryoptionalOK, nil, []string{"\"hello\"", "\"\""}},
		{"ordinarynullable", client.NewOrdinarynullableHttpordinarynullableOK, rpcclient.NewOrdinarynullableJsonrpcordinarynullableOK, nil, []string{"\"hello\"", "null"}},
		{"ordinarynamedoptional", client.NewOrdinarynamedoptionalHttpordinarynamedoptionalOK, rpcclient.NewOrdinarynamedoptionalJsonrpcordinarynamedoptionalOK, nil, []string{"\"hello\"", "\"\""}},
		{"whole", client.NewWholeHttpwholeOK, rpcclient.NewWholeJsonrpcwholeOK, svc.NewViewedHttpwhole, []string{"\"hello\"", "\"\""}},
	}
}

func construct(t *testing.T, fn any, wire string) any {
	t.Helper()
	value := reflect.ValueOf(fn)
	input := reflect.New(value.Type().In(0))
	require.NoError(t, json.Unmarshal([]byte(wire), input.Interface()))
	return value.Call([]reflect.Value{input.Elem()})[0].Interface()
}

func assertJSON(t *testing.T, expected string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, expected, string(raw))
}

func TestConstructors(t *testing.T) {
	for _, tc := range cases() {
		for _, wire := range tc.values {
			t.Run(tc.name+"/"+wire, func(t *testing.T) {
				body := wire
				if tc.name == "whole" {
					body = "{\"value\":" + wire + "}"
				}
				for _, constructor := range []any{tc.httpConstructor, tc.rpcConstructor} {
					result := construct(t, constructor, body)
					assertJSON(t, "{\"value\":"+wire+"}", result)
				}
			})
		}
	}
	require.Error(t, views.ValidateHttprequiredView(&views.HttprequiredView{}))
	require.NoError(t, views.ValidateHttprequiredView(client.NewRequiredHttprequiredOK("")))
}

func projected(t *testing.T, tc storageCase, wire string) any {
	t.Helper()
	if tc.project == nil {
		if wire == "absent" {
			return reflect.New(reflect.TypeOf(tc.httpConstructor).Out(0).Elem()).Interface()
		}
		return construct(t, tc.httpConstructor, wire)
	}
	factory := reflect.ValueOf(tc.project)
	input := reflect.New(factory.Type().In(0).Elem())
	object := "{\"value\":" + wire + "}"
	if wire == "absent" {
		object = "{}"
	}
	require.NoError(t, json.Unmarshal([]byte(object), input.Interface()))
	result := factory.Call([]reflect.Value{input, reflect.ValueOf("default")})
	require.True(t, result[1].IsNil(), "projection validation: %v", result[1].Interface())
	return result[0].Interface()
}

func TestMountedHTTP(t *testing.T) {
	for _, tc := range cases() {
		for _, wire := range tc.values {
			t.Run(tc.name+"/"+wire, func(t *testing.T) {
				result := projected(t, tc, wire)
				calls := 0
				endpoint := loom.Endpoint(func(context.Context, any) (any, error) {
					calls++
					return result, nil
				})
				endpoints := &svc.Endpoints{}
				name := strings.ToUpper(tc.name[:1]) + tc.name[1:]
				reflect.ValueOf(endpoints).Elem().FieldByName(name).Set(reflect.ValueOf(endpoint))
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				recorder := httptest.NewRecorder()
				mux.ServeHTTP(recorder, httptest.NewRequest("GET", "/"+tc.name, nil))
				require.Equal(t, 200, recorder.Code, recorder.Body.String())
				expected := wire
				if tc.name == "whole" {
					expected = "{\"value\":" + wire + "}"
				}
				require.JSONEq(t, expected, recorder.Body.String())
				require.Equal(t, 1, calls)
				srv := httptest.NewServer(mux)
				defer srv.Close()
				generated := client.NewClient("http", strings.TrimPrefix(srv.URL, "http://"), srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				invoke := reflect.ValueOf(generated).MethodByName(name).Call(nil)[0].Interface().(loom.Endpoint)
				decoded, err := invoke(context.Background(), nil)
				require.NoError(t, err)
				require.Equal(t, 2, calls)
				assertJSON(t, "{\"value\":"+wire+"}", decoded)
			})
		}
	}
}

func TestOptionalNamedSourceAbsence(t *testing.T) {
	checked := 0
	for _, tc := range cases() {
		if tc.name != "namedoptional" && tc.name != "ordinarynamedoptional" {
			continue
		}
		checked++
		result := projected(t, tc, "absent")
		calls := 0
		endpoint := loom.Endpoint(func(context.Context, any) (any, error) {
			calls++
			return result, nil
		})
		endpoints := &svc.Endpoints{}
		name := strings.ToUpper(tc.name[:1]) + tc.name[1:]
		reflect.ValueOf(endpoints).Elem().FieldByName(name).Set(reflect.ValueOf(endpoint))
		mux := loomhttp.NewMuxer()
		server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
		response := httptest.NewRecorder()
		require.NotPanics(t, func() {
			mux.ServeHTTP(response, httptest.NewRequest("GET", "/"+tc.name, nil))
		})
		require.Equal(t, 1, calls)
		require.Equal(t, 200, response.Code)
		require.JSONEq(t, "null", response.Body.String())
	}
	require.Equal(t, 2, checked)
}

func TestMountedJSONRPC(t *testing.T) {
	projections := map[string]any{
		"required":         rpcsvc.NewViewedJsonrpcrequired,
		"optional":         rpcsvc.NewViewedJsonrpcoptional,
		"defaulted":        rpcsvc.NewViewedJsonrpcdefaulted,
		"named":            rpcsvc.NewViewedJsonrpcnamed,
		"namedoptional":    rpcsvc.NewViewedJsonrpcnamedoptional,
		"nullable":         rpcsvc.NewViewedJsonrpcnullable,
		"nullableoptional": rpcsvc.NewViewedJsonrpcnullableoptional,
		"number":           rpcsvc.NewViewedJsonrpcnumber,
		"custom":           rpcsvc.NewViewedJsonrpccustom,
		"array":            rpcsvc.NewViewedJsonrpcarray,
		"bytes":            rpcsvc.NewViewedJsonrpcbytes,
		"whole":            rpcsvc.NewViewedJsonrpcwhole,
	}
	for _, tc := range cases() {
		tc.httpConstructor = tc.rpcConstructor
		tc.project = projections[tc.name]
		values := append([]string{}, tc.values...)
		if tc.name == "namedoptional" || tc.name == "ordinarynamedoptional" {
			values = append(values, "absent")
		}
		for _, wire := range values {
			t.Run(tc.name+"/"+wire, func(t *testing.T) {
				result := projected(t, tc, wire)
				calls := 0
				endpoint := loom.Endpoint(func(context.Context, any) (any, error) {
					calls++
					return result, nil
				})
				endpoints := &rpcsvc.Endpoints{}
				name := strings.ToUpper(tc.name[:1]) + tc.name[1:]
				reflect.ValueOf(endpoints).Elem().FieldByName(name).Set(reflect.ValueOf(endpoint))
				mux := loomhttp.NewMuxer()
				rpcserver.Mount(mux, rpcserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
				response := httptest.NewRecorder()
				request := httptest.NewRequest("POST", "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":\"probe\",\"method\":\""+tc.name+"\"}"))
				request.Header.Set("Content-Type", "application/json")
				mux.ServeHTTP(response, request)
				require.Equal(t, 200, response.Code, response.Body.String())
				var envelope map[string]jsontext.Value
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
				require.JSONEq(t, "\"probe\"", string(envelope["id"]))
				expected := wire
				if wire == "absent" {
					expected = "null"
				} else if tc.name == "whole" {
					expected = "{\"value\":" + wire + "}"
				}
				require.JSONEq(t, expected, string(envelope["result"]))
				require.Equal(t, 1, calls)
				if wire == "absent" {
					return
				}
				host := httptest.NewServer(mux)
				defer host.Close()
				generated := rpcclient.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				invoke := reflect.ValueOf(generated).MethodByName(name).Call(nil)[0].Interface().(loom.Endpoint)
				decoded, err := invoke(context.Background(), nil)
				require.NoError(t, err)
				assertJSON(t, "{\"value\":"+wire+"}", decoded)
				require.Equal(t, 2, calls)
			})
		}
	}
	_, err := rpcsvc.NewViewedJsonrpcrequired(&rpcsvc.Jsonrpcrequired{Value: "hello"}, "unknown")
	require.Error(t, err)
}
`
