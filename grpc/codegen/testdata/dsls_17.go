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

// PayloadShapesDSL declares methods named after the named array, named map
// and primitive alias that they use as payload and result, so that the rpc
// and its messages have the same name, a result type used only by gRPC, an
// empty object used as unary and streaming payload and result, and a type
// with a struct:name:proto name used for two errors of a unary and a server
// streaming method.
var PayloadShapesDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
	})
	var Tags = Type("Tags", ArrayOf(String))
	var Index = Type("Index", MapOf(String, Leaf))
	var ID = Type("ID", String)
	var Nothing = Type("Nothing", func() {})
	var Detail = ResultType("application/vnd.detail", func() {
		Attributes(func() {
			Field(1, "name", String)
		})
	})
	var Fault = Type("Fault", func() {
		Field(1, "msg", String)
		ErrorName(2, "name", String)
		Required("name")
		Meta("struct:name:proto", "FaultProto")
	})
	Service("shapes", func() {
		Method("tags", func() {
			Payload(Tags)
			Result(Tags)
			GRPC(func() {})
		})
		Method("index", func() {
			Payload(Index)
			Result(Index)
			GRPC(func() {})
		})
		Method("id", func() {
			Payload(ID)
			Result(ID)
			GRPC(func() {})
		})
		Method("describe", func() {
			Payload(String)
			Result(Detail)
			GRPC(func() {})
		})
		Method("empty", func() {
			Payload(Nothing)
			Result(Nothing)
			GRPC(func() {})
		})
		Method("watch", func() {
			Payload(Nothing)
			StreamingResult(Nothing)
			GRPC(func() {})
		})
		Method("upload", func() {
			StreamingPayload(Nothing)
			Result(Nothing)
			GRPC(func() {})
		})
		Method("fail", func() {
			Payload(String)
			Result(String)
			Error("missing", Fault)
			Error("invalid", Fault)
			GRPC(func() {
				Response("missing", CodeNotFound)
				Response("invalid", CodeInvalidArgument)
			})
		})
		Method("fail_stream", func() {
			Payload(String)
			StreamingResult(String)
			Error("missing", Fault)
			Error("invalid", Fault)
			GRPC(func() {
				Response("missing", CodeNotFound)
				Response("invalid", CodeInvalidArgument)
			})
		})
	})
}
