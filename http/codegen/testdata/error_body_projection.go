package testdata

import . "github.com/CaliLuke/loom/dsl"

// ErrorBodyProjectionDSL covers selected error fields and explicit object bodies.
func ErrorBodyProjectionDSL() {
	object := Type("ErrorDetails", func() {
		Attribute("message", String)
		Required("message")
	})
	cases := []struct {
		name, contentType           string
		body, errorType             any
		optional, renamed, explicit bool
	}{
		{name: "html", contentType: "text/html; charset=utf-8", body: String},
		{name: "plain", contentType: "text/plain", body: String},
		{name: "any", contentType: "application/json", body: Any},
		{name: "map", contentType: "application/json", body: MapOf(String, Any)},
		{name: "object", contentType: "application/json", body: object},
		{name: "explicit", contentType: "application/json", body: String, explicit: true},
		{name: "renamed", contentType: "text/plain", body: String, renamed: true},
		{name: "optional", contentType: "text/plain", body: String, optional: true},
	}
	for i := range cases {
		c := &cases[i]
		c.errorType = Type(c.name+"Error", func() {
			ErrorName("kind", String)
			Attribute("body", c.body, func() {
				if c.renamed {
					Meta("struct:field:name", "Content")
				}
			})
			Required("kind")
			if !c.optional {
				Required("body")
			}
		})
	}
	Service("projection", func() {
		for _, c := range cases {
			Method(c.name, func() {
				Error(c.name, c.errorType)
				HTTP(func() {
					GET("/" + c.name)
					Response(c.name, StatusInternalServerError, func() {
						ContentType(c.contentType)
						Header("kind:X-Kind")
						if c.explicit {
							Body(func() {
								Attribute("body")
							})
						} else {
							Body("body")
						}
					})
				})
			})
		}
	})
}
