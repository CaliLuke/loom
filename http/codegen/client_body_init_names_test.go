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

// TestClientBodyInitNamesPerSource checks that the client functions that
// build bodies of one Go type from different payload types, or from
// different attributes of one payload type, get names of their own, while
// the bodies built from one source share a function.
func TestClientBodyInitNamesPerSource(t *testing.T) {
	cases := []struct {
		method string
		name   string
		arg    string
		code   string
	}{
		{method: "one", name: "NewItemRequestBody", arg: "named.L1", code: "range p {"},
		{method: "two", name: "NewItemRequestBody2", arg: "named.L2", code: "range p {"},
		{method: "one again", name: "NewItemRequestBody", arg: "named.L1", code: "range p {"},
		{method: "stream one", name: "NewItem", arg: "named.L1", code: "range p {"},
		{method: "stream two", name: "NewItem2", arg: "named.L2", code: "range p {"},
		{method: "first", name: "NewItemRequestBodyRequestBody", arg: "*named.Pair", code: "range p.A {"},
		{method: "second", name: "NewItemRequestBodyRequestBody2", arg: "*named.Pair", code: "range p.B {"},
	}
	root := RunHTTPDSL(t, clientBodyInitNamesPerSourceDSL)
	data := CreateHTTPServices(root).Get("named")
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
			require.Len(t, body.Init.ClientArgs, 1)
			require.Equal(t, c.arg, body.Init.ClientArgs[0].TypeRef)
			require.Contains(t, body.Init.ClientCode, c.code)
		})
	}
}

func clientBodyInitNamesPerSourceDSL() {
	item := Type("Item", func() {
		Attribute("name", String)
		Required("name")
	})
	l1 := Type("L1", ArrayOf(item))
	l2 := Type("L2", ArrayOf(item), func() {
		MinLength(1)
	})
	pair := Type("Pair", func() {
		Attribute("a", ArrayOf(item))
		Attribute("b", ArrayOf(item))
		Required("a", "b")
	})
	Service("named", func() {
		unary := func(name, path string, payload any, body string) {
			Method(name, func() {
				Payload(payload)
				HTTP(func() {
					POST(path)
					if body != "" {
						Body(body)
					}
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
		unary("one", "/one", l1, "")
		unary("two", "/two", l2, "")
		unary("one again", "/one-again", l1, "")
		stream("stream one", "/stream-one", l1)
		stream("stream two", "/stream-two", l2)
		unary("first", "/first", pair, "a")
		unary("second", "/second", pair, "b")
	})
}
