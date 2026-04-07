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
