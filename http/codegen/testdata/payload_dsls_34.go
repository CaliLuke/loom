package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathArrayStringValidateDSL = func() {
	Service("ServicePathArrayStringValidate", func() {
		Method("MethodPathArrayStringValidate", func() {
			Payload(func() {
				Attribute("p", ArrayOf(String), func() {
					Enum([]string{"val"})
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


