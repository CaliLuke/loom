package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyHeaderUserDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("a", String)
		Attribute("b", String)
	})
	Service("ServiceBodyHeaderUser", func() {
		Method("MethodBodyHeaderUser", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("b")
				})
			})
		})
	})
}


