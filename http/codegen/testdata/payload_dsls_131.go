package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartArrayTypeDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Attribute("c", MapOf(Int, ArrayOf(String)))
		Required("a", "c")
	})
	Service("ServiceMultipartArrayType", func() {
		Method("MethodMultipartArrayType", func() {
			Payload(ArrayOf(PayloadType))
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


