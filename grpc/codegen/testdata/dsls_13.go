package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// ArrayAliasDSL uses named arrays of a primitive, of a user type and of
// another named array as optional and required message fields, as array
// elements and map values, as branches of a required constructor OneOf passed
// to Field and as branches of a OneOf block next to an inline array. Tags is
// both a field and a branch of the same message and directly the payload and
// result of a method, and so is More. Every named array is the message that wraps the array
// in its "field" attribute.
var ArrayAliasDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Tags = Type("Tags", ArrayOf(String))
	var Leaves = Type("Leaves", ArrayOf(Leaf))
	var More = Type("More", Tags)
	var Envelope = Type("Envelope", func() {
		Field(1, "labels", Tags)
		Field(2, "required_labels", Tags)
		Field(3, "leaf_list", Leaves)
		Field(4, "more", More)
		Field(5, "label_lists", ArrayOf(Tags))
		Field(6, "labels_by_key", MapOf(String, Tags))
		Field(7, "pick", OneOf(Tags, Leaves))
		OneOf("detail", func() {
			Field(9, "names", Tags)
			Field(10, "words", ArrayOf(String))
			Field(11, "count", Int)
		})
		Required("required_labels", "pick")
	})
	Service("arrayalias", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
		Method("list", func() {
			Payload(Tags)
			Result(Tags)
			GRPC(func() {})
		})
		Method("extend", func() {
			Payload(More)
			Result(More)
			GRPC(func() {})
		})
	})
}

// UnionBranchUnionDSL uses unions as branches of other unions: a named union
// as a branch of a named union, of a constructor OneOf passed to Field and of
// a OneOf block, and a constructor OneOf as a branch of a constructor OneOf.
// The named unions are also fields and the direct payloads and results of
// methods. A union branch is the message that wraps the oneof of the union,
// the message of a union payload or result, while a union field remains a
// oneof of the message that holds it.
var UnionBranchUnionDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Other = Type("Other", func() {
		Field(1, "count", Int)
	})
	var Extra = Type("Extra", func() {
		Field(1, "flag", Boolean)
	})
	var Choice = Type("Choice", OneOf(Leaf, Other))
	var Outer = Type("Outer", OneOf(Choice, Extra))
	var Holder = Type("Holder", func() {
		Field(1, "pick", Choice)
		Field(3, "outer", Outer)
		Required("outer")
	})
	var Envelope = Type("Envelope", func() {
		Field(1, "id", String)
		Field(2, "wrapped", OneOf(Choice, Extra))
		Field(4, "nested", OneOf(Leaf, OneOf(Extra, Other)))
		OneOf("block", func() {
			Field(6, "picked", Choice)
			Field(7, "text", String)
		})
		Field(8, "holder", Holder)
		Field(9, "holders", ArrayOf(Holder))
	})
	Service("nestedunion", func() {
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			GRPC(func() {})
		})
		Method("wrap", func() {
			Payload(Outer)
			Result(Outer)
			GRPC(func() {})
		})
		Method("pick", func() {
			Payload(Choice)
			Result(Holder)
			GRPC(func() {})
		})
	})
}

// RecursiveArrayAliasDSL declares a type that reaches itself through a named
// array used as a message field, as a branch of a constructor OneOf, and
// through a named union branch whose branches hold the type and the named
// array. Each cycle crosses a transform helper call.
var RecursiveArrayAliasDSL = func() {
	var Node = Type("Node", func() {
		Field(1, "id", String)
		Field(2, "children", "Nodes")
		Field(3, "kids", OneOf(Int, "Nodes"))
		Field(5, "next", OneOf(Boolean, "Link"))
	})
	Type("Nodes", ArrayOf(Node))
	Type("Link", OneOf(Node, "Nodes"))
	Service("recursivearray", func() {
		Method("walk", func() {
			Payload(Node)
			Result(Node)
			GRPC(func() {})
		})
	})
}

// RecursiveArrayMessageDSL declares a type that reaches itself through a
// named array used as a field, as array elements and as map values, and uses
// the named array directly as a unary payload and result, as a streaming
// result, and as the result of a method with a streaming payload.
var RecursiveArrayMessageDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Node = Type("Node", func() {
		Field(1, "id", String)
		Field(2, "kids", "Nodes")
		Field(3, "grid", ArrayOf("Nodes"))
		Field(4, "index", MapOf(String, "Nodes"))
	})
	var Nodes = Type("Nodes", ArrayOf(Node))
	Service("recursivemessage", func() {
		Method("echo", func() {
			Payload(Node)
			Result(Node)
			GRPC(func() {})
		})
		Method("list", func() {
			Payload(Nodes)
			Result(Nodes)
			GRPC(func() {})
		})
		Method("watch", func() {
			StreamingResult(Nodes)
			GRPC(func() {})
		})
		Method("upload", func() {
			StreamingPayload(Leaf)
			Result(Nodes)
			GRPC(func() {})
		})
	})
}
