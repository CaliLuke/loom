package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathObjectDSL = func() {
	Service("ServicePathObject", func() {
		Method("MethodPathObject", func() {
			Payload(func() {
				Attribute("id")
			})
			HTTP(func() {
				PUT("/{id}")
				Params(func() {
					Param("id")
					Required("id")
				})
			})
		})
	})
}


