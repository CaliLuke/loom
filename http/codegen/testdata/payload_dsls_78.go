package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionUserValidateDSL = func() {
	var SomeType = Type("SomeType", func() {
		Attribute("a", String)
		Required("a")
	})
	var SomeOtherType = Type("SomeOtherType", func() {
		Attribute("b", String)
		Required("b")
	})
	var Union = Type("UnionUserValidate", func() {
		OneOf("Values", func() {
			Attribute("SomeType", SomeType)
			Attribute("SomeOtherType", SomeOtherType)
		})
	})
	Service("ServiceBodyUnionUserValidate", func() {
		Method("MethodBodyUnionUserValidate", func() {
			Payload(func() {
				Attribute("a", Union)
				Required("a")
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


