package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryPathObjectValidateDSL = func() {
	Service("ServiceBodyQueryPathObjectValidate", func() {
		Method("MethodBodyQueryPathObjectValidate", func() {
			Payload(func() {
				Attribute("a", String, func() {
					Pattern("patterna")
				})
				Attribute("b", String, func() {
					Pattern("patternb")
				})
				Attribute("c", String, func() {
					Pattern("patternc")
				})
				Required("a", "b", "c")
			})
			HTTP(func() {
				POST("/{c}")
				Param("b")
			})
		})
	})
}


