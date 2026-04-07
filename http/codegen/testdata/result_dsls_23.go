package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultMultipleViewsTagDSL = func() {
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
	Service("ServiceTagMultipleViews", func() {
		Method("MethodTagMultipleViews", func() {
			Result(ResultType)
			HTTP(func() {
				GET("/")
				Response(StatusAccepted, func() {
					Header("c")
					Tag("b", "value")
				})
				Response(StatusOK)
			})
		})
	})
}


