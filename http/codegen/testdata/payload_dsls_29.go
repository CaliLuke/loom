package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathStringDSL = func() {
	Service("ServicePathString", func() {
		Method("MethodPathString", func() {
			Payload(func() {
				Attribute("p", String)
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


