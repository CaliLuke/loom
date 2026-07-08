package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.

var PayloadCookieStringDSL = func() {
	Service("ServiceCookieString", func() {
		Method("MethodCookieString", func() {
			Payload(func() {
				Attribute("c", String)
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}

var PayloadCookieIntDSL = func() {
	Service("ServiceCookieInt", func() {
		Method("MethodCookieInt", func() {
			Payload(func() {
				Attribute("c", Int)
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}
