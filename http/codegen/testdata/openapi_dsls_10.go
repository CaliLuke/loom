package testdata

import . "github.com/CaliLuke/loom/dsl"


var NotGenerateAttributeDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})
	var PayloadT = Type("Payload", func() {
		Attribute("int", Int, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("string", String, func() {
			Example("")
		})
		Attribute("required_int", Int, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("required_string", String, func() {
			Example("")
		})
		Required("required_int", "required_string")
	})
	var ResultT = Type("Result", func() {
		Attribute("int", Int, func() {
			Example(0)
		})
		Attribute("string", String, func() {
			Meta("openapi:generate", "false")
		})
		Attribute("required_int", Int, func() {
			Example(0)
		})
		Attribute("required_string", String, func() {
			Meta("openapi:generate", "false")
		})
		Required("required_int", "required_string")
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


