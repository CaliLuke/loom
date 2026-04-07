package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var EndpointMultipartConstructorUnion = func() {
	TextPayload := Type("MultipartUnionTextPayload", func() {
		Attribute("text", String)
	})
	JSONPayload := Type("MultipartUnionJSONPayload", func() {
		Attribute("message", String)
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(OneOf(TextPayload, JSONPayload))
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}
