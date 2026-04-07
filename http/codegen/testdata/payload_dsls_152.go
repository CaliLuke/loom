package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var QueryArrayNestedAliasValidateDSL = func() {
	var Float64Alias = Type("Float64Alias", Float64, func() {
		Minimum(10)
	})
	var ArrayAlias = Type("ArrayAlias", ArrayOf(Float64Alias))
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


