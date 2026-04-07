package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryObjectDSL = func() {
	Service("ServiceBodyQueryObject", func() {
		Method("MethodBodyQueryObject", func() {
			Payload(func() {
				Attribute("a", String)
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
				Param("b")
			})
		})
	})
}


