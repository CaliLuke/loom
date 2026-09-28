package testdata

import . "github.com/CaliLuke/loom/dsl"

// PendingUnionDSL exercises named union branches whose canonical types are
// declared later, along with supported recursion through an object field.
var PendingUnionDSL = func() {
	API("pending", func() {
		JSONRPC(func() {})
	})
	inner := Type("Inner", OneOf("Later", Int))
	outer := Type("Outer", OneOf(inner, String))
	Type("Later", func() {
		Field(1, "value", String)
		Required("value")
	})
	node := Type("Node", func() {
		Field(1, "next", OneOf("Node", "Later"))
	})
	Service("pending", func() {
		Method("forward", func() {
			Payload(outer)
			Result(outer)
			HTTP(func() {
				POST("/forward")
			})
			GRPC(func() {})
		})
		Method("recursive", func() {
			Payload(node)
			Result(node)
			HTTP(func() {
				POST("/recursive")
			})
			GRPC(func() {})
		})
	})
	Service("pendingrpc", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("forward", func() {
			Payload(outer)
			Result(outer)
			JSONRPC(func() {})
		})
		Method("recursive", func() {
			Payload(node)
			Result(node)
			JSONRPC(func() {})
		})
	})
}
