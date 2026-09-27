package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.

var PayloadPathCustomIntDSL = func() {
	Service("ServicePathCustomInt", func() {
		Method("MethodPathCustomInt", func() {
			Payload(func() {
				Attribute("p", Int, func() {
					Meta("struct:field:type", "custom.Int", "github.com/CaliLuke/loom/http/codegen/testdata/custom")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}
