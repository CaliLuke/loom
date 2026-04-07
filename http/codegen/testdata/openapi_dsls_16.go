package testdata

import . "github.com/CaliLuke/loom/dsl"


var AdditionalPropertiesEmbeddedPayloadResultDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(func() {
				Attribute("string", String, func() {
					Example("")
				})
				Meta("openapi:additionalProperties", "false")
			})
			Result(func() {
				Attribute("string", String, func() {
					Example("")
				})
				Meta("openapi:additionalProperties", "false")
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}


