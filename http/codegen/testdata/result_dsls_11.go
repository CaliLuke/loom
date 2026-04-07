package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyArrayUserDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("a", String, func() {
			Pattern("apattern")
		})
	})
	Service("ServiceBodyArrayUser", func() {
		Method("MethodBodyArrayUser", func() {
			Result(func() {
				Attribute("b", ArrayOf(ResultType))
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


