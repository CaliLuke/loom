package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	testcodegen "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// TestBuildEndpointNormalizesStreamingBody checks that the WebSocket
// streaming body of the IR is normalized like the request body: a named
// non-object type, including a named union, is replaced with its underlying
// type, while an object body keeps its user type. The evaluated streaming
// body and the inbound stream message keep the design type.
func TestBuildEndpointNormalizesStreamingBody(t *testing.T) {
	cases := []struct {
		method   string
		typeName string
		want     func(expr.DataType) bool
	}{
		{method: "union", typeName: "Choice", want: func(dt expr.DataType) bool {
			_, ok := dt.(*expr.Union)
			return ok
		}},
		{method: "array", typeName: "List", want: func(dt expr.DataType) bool {
			_, ok := dt.(*expr.Array)
			return ok
		}},
		{method: "map", typeName: "Index", want: func(dt expr.DataType) bool {
			_, ok := dt.(*expr.Map)
			return ok
		}},
		{method: "primitive", typeName: "Token", want: func(dt expr.DataType) bool {
			return dt == expr.String
		}},
		{method: "object", typeName: "ObjectStreamingBody", want: func(dt expr.DataType) bool {
			ut, ok := dt.(expr.UserType)
			return ok && ut.Name() == "ObjectStreamingBody"
		}},
	}
	root := testcodegen.RunDSL(t, streamingBodyNormalizationDSL)
	endpoints := root.API.HTTP.Services[0].HTTPEndpoints
	require.Len(t, endpoints, len(cases))
	for i, c := range cases {
		t.Run(c.method, func(t *testing.T) {
			endpoint := endpoints[i]
			require.Equal(t, c.method, endpoint.Name())
			ir := transportir.BuildEndpoint(endpoint)
			require.NotNil(t, ir.Request.StreamingBody)
			require.True(t, c.want(ir.Request.StreamingBody.Type), "streaming body type %T", ir.Request.StreamingBody.Type)

			evaluated, ok := endpoint.StreamingBody.Type.(expr.UserType)
			require.True(t, ok, "evaluated streaming body type %T", endpoint.StreamingBody.Type)
			require.Equal(t, c.typeName, evaluated.Name())
			message, ok := ir.Stream.RequestMessage.Type.(expr.UserType)
			require.True(t, ok, "request message type %T", ir.Stream.RequestMessage.Type)
			require.Equal(t, c.typeName, message.Name())
		})
	}
}

func streamingBodyNormalizationDSL() {
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	payloads := []any{
		dsl.Type("Choice", dsl.OneOf(leaf, other)),
		dsl.Type("List", dsl.ArrayOf(leaf)),
		dsl.Type("Index", dsl.MapOf(dsl.String, leaf)),
		dsl.Type("Token", dsl.String, func() {
			dsl.MinLength(1)
		}),
		dsl.Type("Object", func() {
			dsl.Attribute("leaf", leaf)
		}),
	}
	dsl.Service("svc", func() {
		for i, method := range []string{"union", "array", "map", "primitive", "object"} {
			dsl.Method(method, func() {
				dsl.StreamingPayload(payloads[i])
				dsl.StreamingResult(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/" + method)
				})
			})
		}
	})
}
