package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyInlineObjectDefaultDSL = func() {
	Service("ServiceBodyInlineObject", func() {
		Method("MethodBodyInlineObject", func() {
			Payload(func() {
				Attribute("a", String, func() {
					Default("foo")
				})
			})
			HTTP(func() {
				POST("/")
				Body(func() {
					Attribute("a")
				})
			})
		})
	})

}


