package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var WithHeadersBlockViewedResultDSL = func() {
	var AResult = ResultType("application/vnd.loom.aresult", func() {
		TypeName("AResult")
		Attributes(func() {
			Attribute("required", Int)
			Attribute("optional", Float32)
			Attribute("optional_but_required", UInt)
			Required("required")
		})
		View("tiny", func() {
			Attribute("required")
			Attribute("optional")
			Attribute("optional_but_required")
		})
	})
	Service("ServiceWithHeadersBlockViewedResult", func() {
		Method("MethodA", func() {
			Result(AResult)
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Headers(func() {
						Header("required:X-Request-ID")
						Header("optional:Authorization")
						Header("optional_but_required:Location")
						Required("optional_but_required")
					})
				})
			})
		})
	})
}


