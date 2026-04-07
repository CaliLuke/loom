package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var WithHeadersBlockDSL = func() {
	Service("ServiceWithHeadersBlock", func() {
		Method("MethodA", func() {
			Result(func() {
				Attribute("required", Int)
				Attribute("optional", Float32)
				Attribute("optional_but_required", UInt)
				Required("required")
			})
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


