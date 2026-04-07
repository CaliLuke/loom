package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadFormBodyInvalidDSL = func() {
	Service("ServiceFormBodyInvalid", func() {
		Method("MethodFormBodyInvalid", func() {
			Payload(func() {
				Attribute("count", Int)
				Required("count")
			})
			HTTP(func() {
				POST("/")
				FormRequest()
			})
		})
	})
}


