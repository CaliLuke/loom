package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartUserTypeDSL = func() {
	Service("ServiceMultipartUserType", func() {
		Method("MethodMultipartUserType", func() {
			Payload(func() {
				Attribute("b", String, func() {
					Pattern("patternb")
				})
				Attribute("c", MapOf(Int, ArrayOf(String)))
				Required("b", "c")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


