package testdata

import . "github.com/CaliLuke/loom/dsl"

// ResultViewPresenceDSL exercises the shared response conversion used by unary
// HTTP and WebSocket clients when a view omits fields required by the full type.
// It also checks selected-view output validation across service entry points.
var ResultViewPresenceDSL = func() {
	Interceptor("edit", func() {
		ReadResult(func() {
			Attribute("id")
		})
		WriteResult(func() {
			Attribute("id")
		})
	})
	label := Type("Label", String)
	profile := Type("Profile", func() {
		Attribute("name", String, func() {
			MinLength(3)
		})
		Required("name")
	})
	result := ResultType("application/vnd.presence", func() {
		TypeName("Record")
		Attributes(func() {
			Attribute("id", String, func() {
				MinLength(3)
			})
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
			ServerInterceptor("edit")
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
		Method("list", func() {
			Result(CollectionOf(result))
			HTTP(func() {
				GET("/list")
			})
		})
		Method("tiny", func() {
			Result(result, func() {
				View("tiny")
			})
			HTTP(func() {
				GET("/tiny")
			})
		})
		Method("raw", func() {
			Result(result)
			HTTP(func() {
				POST("/raw")
				SkipRequestBodyEncodeDecode()
			})
		})
	})
}
