package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyObjectOptionalOriginRequestDSL = func() {
	var LinkStart = Type("LinkStart", func() {
		Attribute("continue", String)
	})
	Service("ServiceBodyObjectOptionalOriginRequest", func() {
		Method("MethodBodyObjectOptionalOriginRequest", func() {
			Payload(func() {
				Attribute("body", LinkStart)
			})
			HTTP(func() {
				POST("/")
				Body("body")
				OptionalRequestBody()
			})
		})
	})
}


