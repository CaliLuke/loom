package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUnionValidateDSL = func() {
	var UnionValidate = Type("UnionValidate", func() {
		OneOf("Values", func() {
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	Service("ServiceBodyUnionValidate", func() {
		Method("MethodBodyUnionValidate", func() {
			Payload(func() {
				Attribute("a", UnionValidate)
				Required("a")
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


