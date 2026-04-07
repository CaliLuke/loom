package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var MixedPayloadInBodyDSL = func() {
	var BPayload = Type("BPayload", func() {
		Attribute("int", Int)
		Attribute("bytes", Bytes)
		Required("int")
	})
	var APayload = Type("APayload", func() {
		Attribute("any", Any)
		Attribute("array", ArrayOf(Float32))
		Attribute("map", MapOf(UInt, Any))
		Attribute("object", BPayload)
		Attribute("dup_obj", BPayload)
		Required("array", "object")
	})
	Service("ServiceMixedPayloadInBody", func() {
		Method("MethodA", func() {
			Payload(APayload)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


