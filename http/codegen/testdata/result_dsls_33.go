package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ValidateErrorResponseTypeDSL = func() {
	var AResult = ResultType("application/vnd.loom.aresult", func() {
		TypeName("AResult")
		Attributes(func() {
			Attribute("required", Int)
			Required("required")
		})
	})
	var AError = Type("AError", func() {
		Attribute("error", String)
		Attribute("num_occur", Int, func() {
			Minimum(1)
		})
		Required("error")
	})
	Service("ValidateErrorResponseType", func() {
		Method("MethodA", func() {
			Result(AResult)
			Error("some_error", AError)
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Headers(func() {
						Header("required:X-Request-ID")
					})
				})
				Response("some_error", StatusBadRequest, func() {
					Header("error:X-Application-Error")
					Header("num_occur:X-Occur")
				})
			})
		})
	})
}


