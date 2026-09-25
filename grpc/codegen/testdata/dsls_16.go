package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// MapAliasDSL uses named maps of a primitive, of a user type, of a named
// array and of another named map as optional and required message fields, as
// array elements and as map values. Index is both a field of a message and
// directly the payload and result of a method, and so is More. Both are also
// streaming payloads. Every named map is the message that wraps the map in
// its "field" attribute. Limited carries length and value validations, which
// the message validates.
var MapAliasDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Tags = Type("Tags", ArrayOf(String))
	var Index = Type("Index", MapOf(String, Int))
	var LeafIndex = Type("LeafIndex", MapOf(String, Leaf))
	var TagIndex = Type("TagIndex", MapOf(Int, Tags))
	var More = Type("More", Index)
	var Limited = Type("Limited", MapOf(String, String, func() {
		Elem(func() {
			MinLength(1)
		})
	}), func() {
		MaxLength(2)
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "index", Index)
		Field(2, "required_index", Index)
		Field(3, "leaf_index", LeafIndex)
		Field(4, "tag_index", TagIndex)
		Field(5, "more", More)
		Field(6, "indexes", ArrayOf(Index))
		Field(7, "index_by_key", MapOf(String, Index))
		Field(8, "limited", Limited)
		Required("required_index")
	})
	Service("mapalias", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
		Method("lookup", func() {
			Payload(Index)
			Result(Index)
			GRPC(func() {})
		})
		Method("extend", func() {
			Payload(More)
			Result(More)
			GRPC(func() {})
		})
		Method("upload", func() {
			StreamingPayload(Index)
			Result(Index)
			GRPC(func() {})
		})
		Method("upload_more", func() {
			StreamingPayload(More)
			GRPC(func() {})
		})
	})
}

// RecursiveMapMessageDSL declares a type that reaches itself through a named
// map used as a field, as array elements and as map values, and uses the
// named map directly as a unary payload and result and as a streaming result.
var RecursiveMapMessageDSL = func() {
	var Node = Type("Node", func() {
		Field(1, "id", String)
		Field(2, "kids", "NodeIndex")
		Field(3, "grid", ArrayOf("NodeIndex"))
		Field(4, "index", MapOf(String, "NodeIndex"))
	})
	var NodeIndex = Type("NodeIndex", MapOf(String, Node))
	Service("recursivemap", func() {
		Method("echo", func() {
			Payload(Node)
			Result(Node)
			GRPC(func() {})
		})
		Method("list", func() {
			Payload(NodeIndex)
			Result(NodeIndex)
			GRPC(func() {})
		})
		Method("watch", func() {
			StreamingResult(NodeIndex)
			GRPC(func() {})
		})
	})
}
