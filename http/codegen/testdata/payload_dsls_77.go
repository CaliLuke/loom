package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionUserDSL = func() {
	var SomeType = Type("SomeType", func() {
		Attribute("a", String)
	})
	var SomeOtherType = Type("SomeOtherType", func() {
		Attribute("b", String)
	})
	var Union = Type("UnionUser", func() {
		OneOf("Values", func() {
			Attribute("SomeType", SomeType)
			Attribute("SomeOtherType", SomeOtherType)
		})
	})
	Service("ServiceBodyUnionUser", func() {
		Method("MethodBodyUnionUser", func() {
			Payload(Union)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


