package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyArrayStringValidateDSL = func() {
	Service("ServiceBodyArrayStringValidate", func() {
		Method("MethodBodyArrayStringValidate", func() {
			Payload(func() {
				Attribute("b", ArrayOf(String), func() {
					MinLength(2)
					Elem(func() {
						MinLength(3)
					})
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


