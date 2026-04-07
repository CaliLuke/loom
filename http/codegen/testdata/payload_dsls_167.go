package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionCustomKeysDSL = func() {
	var CustomUnion = Type("CustomUnion", func() {
		OneOf("Values", func() {
			Meta("oneof:type:field", "kind")
			Meta("oneof:value:field", "data")
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	Service("ServiceBodyUnionCustomKeys", func() {
		Method("MethodBodyUnionCustomKeys", func() {
			Payload(CustomUnion)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


