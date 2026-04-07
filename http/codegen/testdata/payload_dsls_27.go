package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadExtendedQueryStringDSL = func() {
	var UT = Type("UserType", func() {
		Attribute("q", String)
	})
	Service("ServiceQueryStringExtendedPayload", func() {
		Method("MethodQueryStringExtendedPayload", func() {
			Payload(func() {
				Extend(UT)
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


