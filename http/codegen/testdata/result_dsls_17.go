package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveArrayUserDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("a", String, func() {
			Pattern("apattern")
		})
	})
	Service("ServiceBodyPrimitiveArrayUser", func() {
		Method("MethodBodyPrimitiveArrayUser", func() {
			Result(ArrayOf(ResultType))
			HTTP(func() {
				POST("/")
			})
		})
	})
}


