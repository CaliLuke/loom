package testdata

import . "github.com/CaliLuke/loom/dsl"


var OpsSocketDSL = func() {
	var _ = API("ops-socket", func() {
		Title("Ops Socket API")
		Description("WebSocket control surface for operator consoles.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("ops-socket", func() {
			Host("api", func() {
				URI("https://api.ops-socket.example.test")
			})
		})
	})
	Service("opsSocket", func() {
		Description("Bidirectional operator control channel.")

		Method("streamCommands", func() {
			NoSecurity()
			Payload(func() {
				Attribute("channel", String, func() {
					Example("deployments")
				})
				Required("channel")
			})
			StreamingPayload(func() {
				Attribute("op", String, func() {
					Example("pause")
				})
				Attribute("target", String, func() {
					Example("worker-eu-1")
				})
				Required("op", "target")
			})
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("command.accepted")
				})
				Attribute("ok", Boolean, func() {
					Example(true)
				})
				Required("event", "ok")
			})
			HTTP(func() {
				GET("/ws/ops/{channel}")
				Param("channel")
			})
		})
	})
}


