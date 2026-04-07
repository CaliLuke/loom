package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryStringSliceDefaultDSL = func() {
	Service("ServiceQueryStringSliceDefault", func() {
		Method("MethodQueryStringSliceDefault", func() {
			Payload(func() {
				Attribute("q", ArrayOf(String), func() {
					Default([]string{"hello", "goodbye"})
				})
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


