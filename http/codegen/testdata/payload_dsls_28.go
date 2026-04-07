package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadExtendedValidateDSL = func() {
	var UT = Type("UserType", func() {
		Attribute("q", String)
		Attribute("h", Int)
		Attribute("body", String)
		Required("h")
	})
	Service("ServiceQueryStringExtendedValidatePayload", func() {
		Method("MethodQueryStringExtendedValidatePayload", func() {
			Payload(func() {
				Extend(UT)
				Required("q", "body")
			})
			HTTP(func() {
				GET("/")
				Param("q")
				Header("h:Location")
			})
		})
	})
}


