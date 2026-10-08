package testdata

import . "github.com/CaliLuke/loom/dsl"

// ResultViewPresenceDSL exercises the shared response conversion used by unary
// HTTP and WebSocket clients when a view omits fields required by the full type.
var ResultViewPresenceDSL = func() {
	label := Type("Label", String)
	profile := Type("Profile", func() {
		Attribute("name", String)
		Required("name")
	})
	result := ResultType("application/vnd.presence", func() {
		TypeName("Record")
		Attributes(func() {
			Attribute("id", String)
			Attribute("details", func() {
				Attribute("name", String)
				Required("name")
			})
			Attribute("label", label)
			Attribute("tags", ArrayOf(String))
			Attribute("counts", MapOf(String, Int))
			Attribute("profile", profile)
			Required("id", "details", "label", "tags", "counts", "profile")
		})
		View("default", func() {
			Attribute("id")
			Attribute("details")
			Attribute("label")
			Attribute("tags")
			Attribute("counts")
			Attribute("profile")
		})
		View("tiny", func() {
			Attribute("id")
		})
	})
	Service("presence", func() {
		Method("show", func() {
			Result(result)
			HTTP(func() {
				GET("/show")
			})
		})
		Method("watch", func() {
			StreamingResult(result)
			HTTP(func() {
				GET("/watch")
			})
		})
	})
}
