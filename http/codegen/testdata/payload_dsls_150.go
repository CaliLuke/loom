package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var QueryMapAliasDSL = func() {
	var MapAlias = Type("MapAlias", MapOf(Float32, Boolean))
	Service("ServiceQueryMapAlias", func() {
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


