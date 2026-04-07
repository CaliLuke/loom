package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ExplicitBodyPrimitiveResultMultipleViewsDSL = func() {
	var ResultType = ResultType("ResultTypeMultipleViews", func() {
		Attribute("a", String, func() {
			MinLength(5)
		})
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
	Service("ServiceExplicitBodyPrimitiveResultMultipleView", func() {
		Method("MethodExplicitBodyPrimitiveResultMultipleView", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
					Body("a")
				})
			})
		})
	})
}


