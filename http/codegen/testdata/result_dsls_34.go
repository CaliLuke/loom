package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyUnionCustomKeysDSL = func() {
	var CustomUnion = Type("CustomUnion", func() {
		OneOf("Values", func() {
			Meta("oneof:type:field", "kind")
			Meta("oneof:value:field", "data")
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	Service("ServiceResultUnionCustomKeys", func() {
		Method("MethodResultUnionCustomKeys", func() {
			Result(CustomUnion)
			HTTP(func() {
				GET("/")
			})
		})
	})
}


