package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var QueryMapAliasValidateDSL = func() {
	var MapAlias = Type("MapAlias", MapOf(Float32, Boolean), func() {
		MinLength(5)
	})
	Service("ServiceQueryMapAliasValidate", func() {
		Method("MethodA", func() {
			Payload(func() {
				Attribute("map", MapAlias)
			})
			HTTP(func() {
				POST("/")
				Params(func() {
					Param("map")
				})
			})
		})
	})
}


