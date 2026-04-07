package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var ParamValidateDSL = func() {
	Service("ServiceParamValidate", func() {
		Method("MethodParamValidate", func() {
			Payload(func() {
				Attribute("a", Int, func() {
					Minimum(1)
				})
			})
			HTTP(func() {
				POST("/")
				Param("a")
			})
		})
	})
}


