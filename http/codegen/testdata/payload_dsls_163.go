package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryCustomNameDSL = func() {
	Service("ServiceQueryCustomName", func() {
		Method("MethodQueryCustomName", func() {
			Payload(func() {
				Attribute("q", String, func() {
					Meta("struct:field:name", "Query")
				})
			})
			HTTP(func() {
				GET("/")
				Params(func() {
					Param("q")
				})
			})
		})
	})
}


