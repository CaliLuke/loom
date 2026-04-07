package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PathIntAliasDSL = func() {
	var IntAlias = Type("IntAlias", Int)
	var Int32Alias = Type("Int32Alias", Int32)
	var Int64Alias = Type("Int64Alias", Int64)
	Service("ServicePathIntAlias", func() {
		Method("MethodA", func() {
			Payload(func() {
				Attribute("int", IntAlias)
				Attribute("int32", Int32Alias)
				Attribute("int64", Int64Alias)
			})
			HTTP(func() {
				POST("/{int}/{int32}/{int64}")
			})
		})
	})
}


