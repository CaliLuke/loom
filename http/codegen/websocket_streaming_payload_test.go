package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"

	. "github.com/CaliLuke/loom/dsl"
)

type (
	// webSocketStreamingPayloadCase is a service with one WebSocket method
	// whose generated code must compile. validates reports whether the
	// server Recv validates the streaming payload. http, when set, adds to
	// the HTTP mapping of the method.
	webSocketStreamingPayloadCase struct {
		service   string
		validates bool
		method    func(types webSocketStreamingPayloadTypes)
		http      func()
	}

	// webSocketStreamingPayloadTypes holds the user types that the cases
	// stream, which the DSL defines before the services.
	webSocketStreamingPayloadTypes struct {
		item, nested, located, required, bounded expr.UserType
	}
)

// TestWebSocketStreamingPayloadValidation checks that the server Recv of a
// WebSocket endpoint validates the streaming payload only with the Validate
// function generated for its body type, and that a body without validations
// gets no validation call.
func TestWebSocketStreamingPayloadValidation(t *testing.T) {
	root := RunHTTPDSL(t, webSocketStreamingPayloadDSL)
	services := CreateHTTPServices(root)
	for _, c := range webSocketStreamingPayloadCases {
		t.Run(c.service, func(t *testing.T) {
			data := services.Get(c.service)
			require.NotNil(t, data)
			require.Len(t, data.Endpoints, 1)
			ws := data.Endpoints[0].ServerWebSocket
			require.NotNil(t, ws)
			require.NotNil(t, ws.Payload)
			validation := serverWebSocketPayloadValidation(ws)
			if !c.validates {
				require.Empty(t, validation)
				require.Empty(t, ws.Payload.ValidateDef)
				return
			}
			require.NotEmpty(t, ws.Payload.ValidateDef)
			require.Equal(t, ws.Payload.ValidateRef, validation)
		})
	}
}

// TestWebSocketStreamingPayloadModuleCompiles generates every WebSocket
// streaming payload case in one module, then compiles and vets it.
func TestWebSocketStreamingPayloadModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, webSocketStreamingPayloadDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/websocketpayload", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
}

var webSocketStreamingPayloadCases = []webSocketStreamingPayloadCase{
	{service: "optional_unary_result", method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.item)
		Result(types.item)
	}},
	{service: "optional_streaming_result", method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.item)
		StreamingResult(types.item)
	}},
	{service: "optional_nested", method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.nested)
		StreamingResult(types.nested)
	}},
	{service: "optional_located", method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.located)
		Result(types.located)
	}},
	{service: "optional_with_param", method: func(types webSocketStreamingPayloadTypes) {
		Payload(func() {
			Attribute("id", String)
		})
		StreamingPayload(types.item)
		StreamingResult(types.item)
	}, http: func() {
		Param("id")
	}},
	{service: "required_field", validates: true, method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.required)
		StreamingResult(types.required)
	}},
	{service: "bounded_field", validates: true, method: func(types webSocketStreamingPayloadTypes) {
		StreamingPayload(types.bounded)
		StreamingResult(types.bounded)
	}},
}

func webSocketStreamingPayloadDSL() {
	inner := Type("Inner", func() {
		Attribute("a", String)
	})
	types := webSocketStreamingPayloadTypes{
		item: Type("Item", func() {
			Attribute("name", String)
			Attribute("note", String)
		}),
		nested: Type("Nested", func() {
			Attribute("name", String)
			Attribute("inner", inner)
		}),
		located: Type("Located", func() {
			Meta("struct:pkg:path", "types")
			Attribute("name", String)
			Attribute("note", String)
		}),
		required: Type("RequiredItem", func() {
			Attribute("name", String)
			Required("name")
		}),
		bounded: Type("BoundedItem", func() {
			Attribute("name", String, func() {
				MinLength(1)
			})
		}),
	}
	for _, c := range webSocketStreamingPayloadCases {
		Service(c.service, func() {
			Method("talk", func() {
				c.method(types)
				HTTP(func() {
					GET("/" + c.service)
					if c.http != nil {
						c.http()
					}
				})
			})
		})
	}
}
