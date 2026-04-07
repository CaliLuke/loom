package testdata

import . "github.com/CaliLuke/loom/dsl"


var NotGenerateHostDSL = func() {
	var _ = API("test", func() {
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
				Meta("openapi:generate", "false")
			})
		})
	})
	Service("testService", func() {
		Method("testEndpoint", func() {
			Result(String)
			HTTP(func() {
				GET("/")
			})
		})
	})
}


