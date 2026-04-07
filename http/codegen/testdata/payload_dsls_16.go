package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryPrimitiveMapStringArrayStringValidateDSL = func() {
	Service("ServiceQueryPrimitiveMapStringArrayStringValidate", func() {
		Method("MethodQueryPrimitiveMapStringArrayStringValidate", func() {
			Payload(MapOf(String, ArrayOf(String)), func() {
				MinLength(1)
				Key(func() {
					Pattern("key")
				})
				Elem(func() {
					MinLength(2)
					Elem(func() {
						Pattern("val")
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


