package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapBoolArrayBoolValidateDSL = func() {
	Service("ServiceQueryMapBoolArrayBoolValidate", func() {
		Method("MethodQueryMapBoolArrayBoolValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, ArrayOf(Boolean)), func() {
					MinLength(1)
					Key(func() {
						Enum(true)
					})
					Elem(func() {
						MinLength(2)
					})
				})
				Required("q")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


