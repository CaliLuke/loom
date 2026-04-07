package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapBoolArrayStringValidateDSL = func() {
	Service("ServiceQueryMapBoolArrayStringValidate", func() {
		Method("MethodQueryMapBoolArrayStringValidate", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, ArrayOf(String)), func() {
					MinLength(1)
					Key(func() {
						Enum(true)
					})
					Elem(func() {
						MinLength(2)
					})
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


