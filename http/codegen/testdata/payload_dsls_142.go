package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var WithParamsAndHeadersBlockDSL = func() {
	Service("ServiceWithParamsAndHeadersBlock", func() {
		Method("MethodA", func() {
			Payload(func() {
				Attribute("required", String)
				Attribute("optional", Int)
				Attribute("optional_but_required_param", Float32)
				Attribute("optional_but_required_header", Float32)
				Attribute("path", UInt)
				Attribute("body", String)
				Required("required")
			})
			HTTP(func() {
				POST("/{path}")
				Params(func() {
					Param("optional", Int)
					Param("optional_but_required_param", Float32)
					Required("optional_but_required_param")
				})
				Headers(func() {
					Header("required", String)
					Header("optional_but_required_header", Float32)
					Required("optional_but_required_header")
				})
			})
		})
	})
}


