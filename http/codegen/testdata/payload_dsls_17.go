package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryPrimitiveMapStringBoolValidateDSL = func() {
	Service("ServiceQueryPrimitiveMapStringBoolValidate", func() {
		Method("MethodQueryPrimitiveMapStringBoolValidate", func() {
			Payload(MapOf(String, Boolean), func() {
				MinLength(1)
				Key(func() {
					Pattern("key")
				})
				Elem(func() {
					Enum(true)
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


