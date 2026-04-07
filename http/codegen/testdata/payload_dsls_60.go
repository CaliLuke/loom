package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadCookieStringDefaultValidateDSL = func() {
	Service("ServiceCookieStringDefaultValidate", func() {
		Method("MethodCookieStringDefaultValidate", func() {
			Payload(func() {
				Attribute("c", String, func() {
					Default("def")
					Enum("def")
				})
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}


