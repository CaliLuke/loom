package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyInlineObjectDSL = func() {
	var ResultType = Type("ResultType", func() {
		Attribute("parent", func() {
			Attribute("child")
		})
	})
	Service("ServiceBodyInlineObject", func() {
		Method("MethodBodyInlineObject", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


