package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathArrayStringDSL = func() {
	Service("ServicePathArrayString", func() {
		Method("MethodPathArrayString", func() {
			Payload(func() {
				Attribute("p", ArrayOf(String))
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


