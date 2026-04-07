package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionDSL = func() {
	var Union = Type("Union", func() {
		OneOf("Values", func() {
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	Service("ServiceBodyUnion", func() {
		Method("MethodBodyUnion", func() {
			Payload(Union)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


