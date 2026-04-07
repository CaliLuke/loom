package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ExplicitBodyUserResultObjectDSL = func() {
	var UserType = Type("UserType", func() {
		Attribute("x", String)
		Attribute("y", Int)
	})
	var ResultType = ResultType("ResultType", func() {
		Attribute("a", UserType)
		Attribute("b", String)
		Attribute("c", String)
	})
	Service("ServiceExplicitBodyUserResultObject", func() {
		Method("MethodExplicitBodyUserResultObject", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
					Header("b:Content-Type")
					Body(func() {
						Attribute("a")
					})
				})
			})
		})
	})
}


