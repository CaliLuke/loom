package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapBoolBoolValidateDSL = func() {
	Service("ServiceQueryMapBoolBoolValidate", func() {
		Method("MethodQueryMapBoolBoolValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, Boolean), func() {
					MinLength(1)
					Key(func() {
						Enum(false)
					})
					Elem(func() {
						Enum(true)
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


