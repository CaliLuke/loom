package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var CustomMessageNameDSL = func() {
	var CustomType = Type("CustomType", func() {
		Meta("struct:name:proto", "CustomType")
		Field(1, "a", Int)
		Field(2, "b", String)
	})
	Service("CustomMessageName", func() {
		Method("Unary", func() {
			Payload(CustomType)
			Result(CustomType)
			GRPC(func() {})
		})
		Method("Stream", func() {
			StreamingPayload(CustomType)
			StreamingResult(CustomType)
			GRPC(func() {})
		})
	})
}

// InvalidFirstCharacterNamesDSL declares types, fields and union values whose
// design names start with a digit, an underscore or a non-ASCII rune, none of
// which may start a protocol buffer identifier.
var InvalidFirstCharacterNamesDSL = func() {
	var Point = Type("3DPoint", func() {
		Field(1, "2x", Int)
		Field(2, "_9lives", String)
		Field(3, "日本name", String)
		Field(4, "élan", Boolean)
		OneOf("9value", func() {
			Field(5, "1st", String)
			Field(6, "2nd", Int)
		})
	})
	Service("InvalidFirstCharacter", func() {
		Method("Unary", func() {
			Payload(Point)
			Result(ArrayOf(Point))
			GRPC(func() {})
		})
	})
}
