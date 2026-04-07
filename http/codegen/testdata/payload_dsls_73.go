package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyObjectOptionalRequestDSL = func() {
	var LinkStart = Type("LinkStart", func() {
		Attribute("continue", String)
	})
	Service("ServiceBodyObjectOptionalRequest", func() {
		Method("MethodBodyObjectOptionalRequest", func() {
			Payload(LinkStart)
			HTTP(func() {
				POST("/")
				Body(LinkStart)
				OptionalRequestBody()
			})
		})
	})
}


