package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryObjectValidateDSL = func() {
	Service("ServiceBodyQueryObjectValidate", func() {
		Method("MethodBodyQueryObjectValidate", func() {
			Payload(func() {
				Attribute("a", String, func() {
					Pattern("patterna")
				})
				Attribute("b", String, func() {
					Pattern("patternb")
				})
				Required("a", "b")
			})
			HTTP(func() {
				POST("/")
				Param("b")
			})
		})
	})
}


