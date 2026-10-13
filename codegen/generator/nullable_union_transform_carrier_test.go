package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNullableObjectUnionTransformCarrier(t *testing.T) {
	runDesignHarness(t, "example.com/carrierprobe", nullableUnionTransformDSL, nullableUnionTransformHarness)
}

func nullableUnionTransformDSL() {
	dsl.API("carrierprobe", func() {
	})
	object := dsl.Type("Object", func() {
		dsl.Attribute("text", dsl.String)
	})
	shapes := []struct {
		name string
		typ  expr.DataType
	}{
		{"object", object},
		{"array", dsl.Type("List", dsl.ArrayOf(dsl.String))},
		{"map", dsl.Type("Dict", dsl.MapOf(dsl.String, dsl.String))},
		{"union", dsl.Type("Nested", dsl.OneOf(dsl.String, dsl.Int))},
		{"primitive", dsl.Type("Text", dsl.String)},
	}
	inherited := make([]expr.DataType, len(shapes))
	for i, shape := range shapes {
		if shape.name == "union" {
			// Intrinsic nullable named union declarations are blocked by #829.
			continue
		}
		inherited[i] = dsl.Type("Nullable"+shape.typ.Name(), shape.typ, func() {
			dsl.Nullable()
		})
	}
	holder := dsl.Type("Holder", func() {
		dsl.OneOf("branch", func() {
			for i, shape := range shapes {
				dsl.Attribute(shape.name+"_ordinary", shape.typ)
				if inherited[i] != nil {
					dsl.Attribute(shape.name+"_intrinsic", inherited[i])
				}
				dsl.Attribute(shape.name+"_occurrence", shape.typ, func() {
					dsl.Nullable()
				})
			}
		})
		dsl.Required("branch")
	})
	dsl.Service("carrierprobe", func() {
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
		})
	})
}

const nullableUnionTransformHarness = `package carrierprobe

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	service "example.com/carrierprobe/gen/carrierprobe"
	client "example.com/carrierprobe/gen/http/carrierprobe/client"
	server "example.com/carrierprobe/gen/http/carrierprobe/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type echo struct {
	calls atomic.Int32
}

func (s *echo) Show(_ context.Context, payload *service.Holder) (*service.Holder, error) {
	s.calls.Add(1)
	return payload, nil
}

func TestAbsentObjectBranchConversion(t *testing.T) {
	for _, intrinsic := range []bool{false, true} {
		payload := &service.Holder{}
		if intrinsic {
			payload.Branch = service.NewBranchObjectIntrinsic(loom.Nullable[service.NullableObject]{})
		} else {
			payload.Branch = service.NewBranchObjectOccurrence(loom.Nullable[service.Object]{})
		}
		request := client.NewShowRequestBody(payload)
		response := server.NewShowResponseBody(payload)
		if intrinsic {
			requestValue, requestSelected := request.Branch.AsObjectIntrinsic()
			responseValue, responseSelected := response.Branch.AsObjectIntrinsic()
			require.True(t, requestSelected)
			require.True(t, responseSelected)
			require.False(t, requestValue.Present())
			require.False(t, responseValue.Present())
		} else {
			requestValue, requestSelected := request.Branch.AsObjectOccurrence()
			responseValue, responseSelected := response.Branch.AsObjectOccurrence()
			require.True(t, requestSelected)
			require.True(t, responseSelected)
			require.False(t, requestValue.Present())
			require.False(t, responseValue.Present())
		}
	}
}

func TestBranchConversionRoundTrips(t *testing.T) {
	implementation := &echo{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(service.NewEndpoints(implementation), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	transport := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	api := service.NewClient(transport.Show())

	for _, shape := range []struct {
		name  string
		value string
	}{
		{"object", "{\"text\":\"value\"}"},
		{"array", "[\"value\"]"},
		{"map", "{\"key\":\"value\"}"},
		{"union", "{\"type\":\"String\",\"value\":\"value\"}"},
		{"primitive", "\"value\""},
	} {
		for _, mode := range []string{"ordinary", "intrinsic", "occurrence"} {
			if shape.name == "union" && mode == "intrinsic" {
				// See #829; direct transform tests retain this carrier case.
				continue
			}
			for _, value := range []string{shape.value, "null", ""} {
				t.Run(shape.name+"/"+mode+"/"+value, func(t *testing.T) {
					branch := shape.name + "_" + mode
					envelope := "{\"type\":\"" + branch + "\""
					if value != "" {
						envelope += ",\"value\":" + value
					}
					wire := "{\"branch\":" + envelope + "}}"
					before := implementation.calls.Load()
					if value == "" || (value == "null" && mode == "ordinary") {
						response, err := host.Client().Post(host.URL+"/show", "application/json", strings.NewReader(wire))
						require.NoError(t, err)
						body, err := io.ReadAll(response.Body)
						require.NoError(t, err)
						require.NoError(t, response.Body.Close())
						require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
						require.Equal(t, before, implementation.calls.Load())
						return
					}
					var payload service.Holder
					require.NoError(t, json.Unmarshal([]byte(wire), &payload))
					result, err := api.Show(context.Background(), &payload)
					require.NoError(t, err)
					require.Equal(t, before+1, implementation.calls.Load())
					encoded, err := json.Marshal(result)
					require.NoError(t, err)
					require.JSONEq(t, wire, string(encoded), "all four HTTP conversion boundaries must preserve the branch carrier")
				})
			}
		}
	}
}
`
