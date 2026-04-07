package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ExplicitBodyResultCollectionDSL = func() {
	var ResultType = ResultType("ResultType", func() {
		Attributes(func() {
			Attribute("x", String, func() {
				MinLength(5)
			})
		})
	})
	Service("ServiceExplicitBodyResultCollection", func() {
		Method("MethodExplicitBodyResultCollection", func() {
			Result(func() {
				Attribute("a", CollectionOf(ResultType))
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Body("a")
				})
			})
		})
	})
}


