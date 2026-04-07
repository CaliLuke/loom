package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyMapStringValidateDSL = func() {
	Service("ServiceBodyMapStringValidate", func() {
		Method("MethodBodyMapStringValidate", func() {
			Payload(func() {
				Attribute("b", MapOf(String, String), func() {
					Elem(func() {
						MinLength(2)
					})
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


