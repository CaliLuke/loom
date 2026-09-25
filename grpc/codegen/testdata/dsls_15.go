package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// CLIProtoJSONDSL declares request messages whose protocol buffer JSON
// differs from the design: attribute names that are not the protocol buffer
// field names, a protocol buffer keyword, nested arrays, maps with string and
// integer keys, bytes, 64-bit integers, Any values, and a named union used as
// a field, in the type of array elements and map values, and directly as a
// payload. The plain method has no union, the send method has unions, and the
// others use wrapper messages.
var CLIProtoJSONDSL = func() {
	var Leaf = Type("Leaf", func() {
		Field(1, "leafName", String)
		Field(2, "message", String)
		Required("leafName")
	})
	var Choice = Type("Choice", OneOf(Leaf, Int))
	var Slot = Type("Slot", func() {
		Field(1, "choice", Choice)
	})
	Service("pbjson", func() {
		Method("plain", func() {
			Payload(func() {
				Field(1, "RequestID", String)
				Field(2, "tags", ArrayOf(ArrayOf(String)))
				Field(3, "by_name", MapOf(String, Leaf))
				Field(4, "by_id", MapOf(Int, String))
				Field(5, "blob", Bytes)
				Field(6, "big", Int64)
				Field(7, "anything", Any)
				Field(8, "ratio", Float32)
				Required("RequestID")
			})
			GRPC(func() {})
		})
		Method("send", func() {
			Payload(func() {
				Field(1, "RequestID", String)
				Field(2, "pick", Choice)
				Field(4, "choices", ArrayOf(Slot))
				Field(5, "choice_map", MapOf(String, Slot))
				Field(6, "leaves", ArrayOf(Leaf))
				Required("RequestID", "pick")
			})
			GRPC(func() {})
		})
		Method("union", func() {
			Payload(Choice)
			GRPC(func() {})
		})
		Method("scalar", func() {
			Payload(String)
			GRPC(func() {})
		})
		Method("list", func() {
			Payload(ArrayOf(Leaf))
			GRPC(func() {})
		})
	})
}
