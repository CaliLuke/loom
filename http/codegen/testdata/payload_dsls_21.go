package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryStringMappedDSL = func() {
	Service("ServiceQueryStringMapped", func() {
		Method("MethodQueryStringMapped", func() {
			Payload(func() {
				Attribute("query")
			})
			HTTP(func() {
				GET("/")
				Param("query:q")
			})
		})
	})
}


