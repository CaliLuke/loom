package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var EmptyBodyResultMultipleViewsDSL = func() {
	var ResultType = ResultType("ResultTypeMultipleViews", func() {
		Attribute("a", String)
		Attribute("b", String)
		Attribute("c", String)
		View("default", func() {
			Attribute("a")
			Attribute("c")
		})
		View("tiny", func() {
			Attribute("c")
		})
	})
	Service("ServiceEmptyBodyResultMultipleView", func() {
		Method("MethodEmptyBodyResultMultipleView", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("c:Location")
					Body(Empty)
				})
			})
		})
	})
}


