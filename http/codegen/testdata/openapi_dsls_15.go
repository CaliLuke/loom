package testdata

import . "github.com/CaliLuke/loom/dsl"


var AdditionalPropertiesPayloadResultDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
	})
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Payload(PayloadT, func() {
				Meta("openapi:additionalProperties", "false")
			})
			Result(ResultT, func() {
				Meta("openapi:additionalProperties", "false")
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}


