package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryPathObjectDSL = func() {
	Service("ServiceBodyQueryPathObject", func() {
		Method("MethodBodyQueryPathObject", func() {
			Payload(func() {
				Attribute("a", String)
				Attribute("b", String)
				Attribute("c", String)
			})
			HTTP(func() {
				POST("/{c}")
				Param("b")
			})
		})
	})
}


