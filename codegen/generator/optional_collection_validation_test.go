package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

func TestOptionalCollectionValidationGenerated(t *testing.T) {
	runDesignHarness(t, "example.com/collections", optionalCollectionValidationDSL, optionalCollectionValidationHarness)
}

func optionalCollectionValidationDSL() {
	dsl.API("collections", func() {
		dsl.JSONRPC(func() {})
	})
	bounds := func() {
		dsl.MinLength(2)
		dsl.MaxLength(4)
	}
	list := dsl.Type("List", dsl.ArrayOf(dsl.String), bounds)
	settings := dsl.Type("Settings", func() {
		dsl.Field(1, "levels", dsl.ArrayOf(dsl.String), bounds)
		dsl.Field(2, "labels", dsl.MapOf(dsl.String, dsl.String), bounds)
		dsl.Field(3, "named", list)
		dsl.Field(4, "required", dsl.ArrayOf(dsl.String), bounds)
		dsl.Required("required")
	})
	arraySettings := dsl.Type("ArraySettings", func() {
		dsl.Attribute("levels", dsl.ArrayOf(dsl.String), bounds)
		dsl.Attribute("named", list)
		dsl.Attribute("required", dsl.ArrayOf(dsl.String), bounds)
		dsl.Required("required")
	})
	absent := dsl.Type("Absent", func() {
		dsl.Attribute("missing", dsl.Boolean)
		dsl.Required("missing")
	})
	choice := dsl.Type("Choice", dsl.OneOf(arraySettings, absent), func() {
		dsl.Untagged()
	})
	dsl.Service("collections", func() {
		dsl.Method("read", func() {
			dsl.Result(choice)
			dsl.HTTP(func() {
				dsl.GET("/read")
			})
		})
		dsl.Method("check", func() {
			dsl.Payload(settings)
			dsl.Result(settings)
			dsl.HTTP(func() {
				dsl.POST("/check")
			})
			dsl.GRPC(func() {})
		})
	})
	dsl.Service("collectionsrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("check", func() {
			dsl.Payload(settings)
			dsl.Result(settings)
			dsl.JSONRPC(func() {})
		})
	})
}

const optionalCollectionValidationHarness = `package collections_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	svc "example.com/collections/gen/collections"
	rpc "example.com/collections/gen/collectionsrpc"
	httpserver "example.com/collections/gen/http/collections/server"
	rpcserver "example.com/collections/gen/jsonrpc/collectionsrpc/server"
)

func TestCollectionPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid, arraysValid bool
	}{
		{"omitted", "{\"required\":[\"a\",\"b\"]}", true, true},
		{"supplied", "{\"required\":[\"a\",\"b\"],\"levels\":[\"a\",\"b\"],\"labels\":{\"a\":\"a\",\"b\":\"b\"},\"named\":[\"a\",\"b\"]}", true, true},
		{"empty array", "{\"required\":[\"a\",\"b\"],\"levels\":[]}", false, false},
		{"short array", "{\"required\":[\"a\",\"b\"],\"levels\":[\"a\"]}", false, false},
		{"long array", "{\"required\":[\"a\",\"b\"],\"levels\":[\"a\",\"b\",\"c\",\"d\",\"e\"]}", false, false},
		{"empty map", "{\"required\":[\"a\",\"b\"],\"labels\":{}}", false, true},
		{"short map", "{\"required\":[\"a\",\"b\"],\"labels\":{\"a\":\"a\"}}", false, true},
		{"long map", "{\"required\":[\"a\",\"b\"],\"labels\":{\"a\":\"a\",\"b\":\"b\",\"c\":\"c\",\"d\":\"d\",\"e\":\"e\"}}", false, true},
		{"empty named", "{\"required\":[\"a\",\"b\"],\"named\":[]}", false, false},
		{"missing required", "{}", false, false},
		{"empty required", "{\"required\":[]}", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var value svc.Settings
			require.NoError(t, json.Unmarshal([]byte(tc.body), &value))
			err := svc.ValidateArraySettings(&svc.ArraySettings{Levels: value.Levels, Named: value.Named, Required: value.Required})
			require.Equal(t, tc.arraysValid, err == nil, "%v", err)
			for _, protocol := range []string{"http", "rpc"} {
				t.Run(protocol, func(t *testing.T) {
					calls := 0
					endpoint := func(_ context.Context, payload any) (any, error) {
						calls++
						encoded, err := json.Marshal(payload)
						require.NoError(t, err)
						var got svc.Settings
						require.NoError(t, json.Unmarshal(encoded, &got))
						require.Equal(t, value, got)
						return payload, nil
					}
					mux := loomhttp.NewMuxer()
					path, body := "/check", tc.body
					if protocol == "http" {
						httpserver.Mount(mux, httpserver.New(&svc.Endpoints{Check: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
					} else {
						rpcserver.Mount(mux, rpcserver.New(&rpc.Endpoints{Check: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
						path = "/rpc"
						body = "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"check\",\"params\":" + tc.body + "}"
					}
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest("POST", path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					mux.ServeHTTP(recorder, request)
					if tc.valid {
						require.Equal(t, 1, calls, recorder.Body.String())
						require.Equal(t, 200, recorder.Code, recorder.Body.String())
					} else {
						require.Zero(t, calls)
						if protocol == "http" {
							require.Equal(t, 400, recorder.Code, recorder.Body.String())
						} else {
							require.Contains(t, recorder.Body.String(), "\"error\":")
						}
					}
				})
			}
		})
	}
}
`
