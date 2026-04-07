package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ExplicitContentTypeResponseDSL = func() {
	var ResultType = ResultType("ResultType", func() {
		Attribute("a", String)
		Attribute("b", String)
	})
	Service("ServiceExplicitContentTypeResponse", func() {
		Method("MethodExplicitContentTypeResponse", func() {
			Result(ResultType)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					ContentType("application/custom+json")
				})
			})
		})
	})
}


