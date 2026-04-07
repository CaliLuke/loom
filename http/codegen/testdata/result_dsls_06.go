package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ExplicitBodyUserResultObjectMultipleViewDSL = func() {
	var UserType = Type("UserType", func() {
		Attribute("x", String)
		Attribute("y", Int)
	})
	var ResultType = ResultType("ResultTypeMultipleViews", func() {
		Attribute("a", UserType)
		Attribute("b", String)
		Attribute("c", String)
		View("default", func() {
			Attribute("a")
			Attribute("b")
			Attribute("c")
		})
		View("tiny", func() {
			Attribute("a")
			Attribute("c")
		})
	})
	Service("ServiceExplicitBodyUserResultObjectMultipleView", func() {
		Method("MethodExplicitBodyUserResultObjectMultipleView", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
					Body(func() {
						Attribute("a")
					})
				})
			})
		})
	})
}


