package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var QueryArrayAliasDSL = func() {
	var ArrayAlias = Type("ArrayAlias", ArrayOf(UInt))
	Service("ServiceQueryArrayAlias", func() {
		Method("MethodA", func() {
			Payload(func() {
				Attribute("array", ArrayAlias)
			})
			HTTP(func() {
				POST("/")
				Params(func() {
					Param("array")
				})
			})
		})
	})
}


