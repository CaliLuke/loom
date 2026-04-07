package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadCookieStringValidateDSL = func() {
	Service("ServiceCookieStringValidate", func() {
		Method("MethodCookieStringValidate", func() {
			Payload(func() {
				Attribute("c", String, func() {
					Pattern("cookie")
				})
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}


