package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadCookieCustomNameDSL = func() {
	Service("ServiceCookieCustomName", func() {
		Method("MethodCookieCustomName", func() {
			Payload(func() {
				Attribute("c", String, func() {
					Meta("struct:field:name", "Cookie")
				})
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}


