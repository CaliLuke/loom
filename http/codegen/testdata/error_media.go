package testdata

import . "github.com/CaliLuke/loom/dsl"

// ErrorMediaDSL covers error representations sharing a status in both orders.
func ErrorMediaDSL() {
	details := Type("ErrorDetails", func() {
		Attribute("message", String)
		Required("message")
	})
	html := Type("HTMLError", func() {
		ErrorName("kind", String)
		Attribute("body", String)
		Required("kind", "body")
	})
	object := Type("ObjectError", func() {
		ErrorName("kind", String)
		Attribute("body", details)
		Required("kind", "body")
	})
	Service("projection", func() {
		for _, name := range []string{"mixed", "reversed", "same_media"} {
			Method(name, func() {
				NoSecurity()
				Error("html", html)
				Error("object", object)
				HTTP(func() {
					GET("/" + name)
					order := []string{"html", "object"}
					if name == "reversed" {
						order = []string{"object", "html"}
					}
					for _, errorName := range order {
						Response(errorName, StatusInternalServerError, func() {
							if errorName == "html" && name != "same_media" {
								ContentType("text/html; charset=utf-8")
							} else {
								ContentType("application/json")
							}
							Header("kind:X-Kind")
							Body("body")
						})
					}
				})
			})
		}
	})
}
