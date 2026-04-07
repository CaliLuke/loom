package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartWithParamDSL = func() {
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
	Service("ServiceMultipartWithParam", func() {
		Method("MethodMultipartWithParam", func() {
			Payload(PayloadType)
			Result(String)
			HTTP(func() {
				POST("/")
				Param("c")
				MultipartRequest()
			})
		})
	})
}


