package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapIntMapStringArrayIntValidateDSL = func() {
	Service("ServiceQueryMapIntMapStringArrayIntValidate", func() {
		Method("MethodQueryMapIntMapStringArrayIntValidate", func() {
			Payload(MapOf(Int, MapOf(String, ArrayOf(Int))), func() {
				Key(func() {
					Enum(1, 2, 3)
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


