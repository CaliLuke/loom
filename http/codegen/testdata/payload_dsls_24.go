package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryStringDefaultValidateDSL = func() {
	Service("ServiceQueryStringDefaultValidate", func() {
		Method("MethodQueryStringDefaultValidate", func() {
			Payload(func() {
				Attribute("q", func() {
					Default("def")
					Enum("def")
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


