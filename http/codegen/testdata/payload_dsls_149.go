package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var QueryArrayAliasValidateDSL = func() {
	var ArrayAlias = Type("ArrayAlias", ArrayOf(UInt), func() {
		MinLength(3)
		Elem(func() {
			Minimum(10)
		})
	})
	Service("ServiceQueryArrayAliasValidate", func() {
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


