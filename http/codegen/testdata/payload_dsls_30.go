package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathStringValidateDSL = func() {
	Service("ServicePathStringValidate", func() {
		Method("MethodPathStringValidate", func() {
			Payload(func() {
				Attribute("p", String, func() {
					Enum("val")
				})
				Required("p")
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


