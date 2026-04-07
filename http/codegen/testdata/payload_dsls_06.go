package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryMapStringArrayBoolDSL = func() {
	Service("ServiceQueryMapStringArrayBool", func() {
		Method("MethodQueryMapStringArrayBool", func() {
			Payload(func() {
				Attribute("q", MapOf(String, ArrayOf(Boolean)))
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


