package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadFormBodyObjectOptionalDSL = func() {
	Service("ServiceFormBodyObjectOptional", func() {
		Method("MethodFormBodyObjectOptional", func() {
			Payload(func() {
				Attribute("client_id", String)
				Attribute("scope", ArrayOf(String))
				Attribute("active", Boolean)
				Required("client_id")
			})
			HTTP(func() {
				POST("/")
				FormRequest()
			})
		})
	})
}


