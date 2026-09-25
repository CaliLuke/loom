package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestClientBodyInitNames checks the names of the client functions that
// build the request bodies and the WebSocket streaming bodies of a service.
// Bodies of one Go type share a function, and a body whose Go type differs
// from that of a body with the same base name, such as a nested collection
// of the element type of a flat collection, gets a function of its own.
func TestClientBodyInitNames(t *testing.T) {
	cases := []struct {
		method string
		name   string
		ref    string
	}{
		{method: "flat", name: "NewItemRequestBody", ref: "[]*ItemRequestBody"},
		{method: "nested", name: "NewItemRequestBody2", ref: "[][]*ItemRequestBody"},
		{method: "flat again", name: "NewItemRequestBody", ref: "[]*ItemRequestBody"},
		{method: "stream flat", name: "NewItem", ref: "[]*Item"},
		{method: "stream nested", name: "NewItem2", ref: "[][]*Item"},
		{method: "stream nested again", name: "NewItem2", ref: "[][]*Item"},
		{method: "stream map", name: "NewMapStringItem", ref: "map[string][]*Item"},
	}
	root := RunHTTPDSL(t, clientBodyInitNamesDSL)
	data := CreateHTTPServices(root).Get("batch")
	require.NotNil(t, data)
	for _, c := range cases {
		t.Run(c.method, func(t *testing.T) {
			ed := data.Endpoint(c.method)
			require.NotNil(t, ed)
			body := ed.Payload.Request.ClientBody
			if ed.ClientWebSocket != nil {
				body = ed.ClientWebSocket.Payload
			}
			require.NotNil(t, body)
			require.NotNil(t, body.Init)
			require.Equal(t, c.name, body.Init.Name)
			require.Equal(t, c.ref, body.Init.ReturnTypeRef)
		})
	}
}

func clientBodyInitNamesDSL() {
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	Service("batch", func() {
		unary := func(name, path string, payload any) {
			Method(name, func() {
				Payload(payload)
				HTTP(func() {
					POST(path)
				})
			})
		}
		stream := func(name, path string, payload any) {
			Method(name, func() {
				StreamingPayload(payload)
				Result(String)
				HTTP(func() {
					GET(path)
				})
			})
		}
		unary("flat", "/flat", ArrayOf(item))
		unary("nested", "/nested", ArrayOf(ArrayOf(item)))
		unary("flat again", "/flat-again", ArrayOf(item))
		stream("stream flat", "/stream-flat", ArrayOf(item))
		stream("stream nested", "/stream-nested", ArrayOf(ArrayOf(item)))
		stream("stream nested again", "/stream-nested-again", ArrayOf(ArrayOf(item)))
		stream("stream map", "/stream-map", MapOf(String, ArrayOf(item)))
	})
}
