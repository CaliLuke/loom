package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapStringMapIntStringValidateDSL = func() {
	Service("ServiceQueryMapStringMapIntStringValidate", func() {
		Method("MethodQueryMapStringMapIntStringValidate", func() {
			Payload(MapOf(String, MapOf(Int, String)), func() {
				Key(func() {
					Enum("foo")
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


