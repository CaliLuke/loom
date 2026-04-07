package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadWithValidatedAliasDSL = func() {
	var ValidatedString = Type("ValidatedString", String, func() {
		MinLength(10)
		Pattern("^[a-zA-Z]+$")
	})

	Service("ServicePayloadValidatedAlias", func() {
		Method("Method", func() {
			StreamingPayload(func() {
				Attribute("name", ValidatedString)
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}


