package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// MappedNamesDSL declares attributes with a mapping suffix, such as "n:m", in
// messages, nested types, unions and streaming messages: optional, required
// and defaulted primitives, validated strings, objects, arrays, maps, a
// constructor OneOf and a OneOf block with suffixed branches. gRPC ignores the
// suffix: the protocol buffer fields and the Go fields of the service and
// protocol buffer types are named after the part that precedes the colon.
var MappedNamesDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "leaf:l", String)
		Field(2, "count:c", Int)
		Required("count:c")
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "n:m", String, func() {
			MinLength(2)
		})
		Field(2, "req:r", Int)
		Field(3, "def:d", Int, func() {
			Default(3)
		})
		Field(4, "pick:p", OneOf(String, Int))
		Field(6, "obj:o", Leaf)
		Field(7, "list:ls", ArrayOf(String))
		Field(8, "index:ix", MapOf(String, Leaf))
		OneOf("choice:ch", func() {
			Field(9, "text:t", String)
			Field(10, "leaf_branch:lb", Leaf)
		})
		Required("req:r", "pick:p", "obj:o")
	})
	Service("mappednames", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
		Method("stream", func() {
			Payload(func() {
				Field(1, "id:i", String)
				Required("id:i")
			})
			StreamingPayload(Leaf)
			StreamingResult(Envelope)
			GRPC(func() {})
		})
	})
}
