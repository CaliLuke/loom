package testdata

import . "github.com/CaliLuke/loom/dsl"


var AdditionalPropertiesTypeDSL = func() {
	var PayloadT = Type("Payload", func() {
		Attribute("string", String, func() {
			Example("")
		})
		Meta("openapi:additionalProperties", "false")
	})
	var ResultT = Type("Result", func() {
		Attribute("string", String, func() {
			Example("")
		})
		Meta("openapi:additionalProperties", "false")
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
			Payload(PayloadT)
			Result(ResultT)
			HTTP(func() {
				GET("/")
			})
		})
	})
}


