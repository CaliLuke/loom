package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapBoolBoolDSL = func() {
	Service("ServiceQueryMapBoolBool", func() {
		Method("MethodQueryMapBoolBool", func() {
			Payload(func() {
				Attribute("q", MapOf(Boolean, Boolean))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


