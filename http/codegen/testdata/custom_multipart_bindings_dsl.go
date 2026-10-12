package testdata

import . "github.com/CaliLuke/loom/dsl"

// CustomMultipartBindingsDSL exercises bound fields beside an opaque file array.
func CustomMultipartBindingsDSL(location string) func() {
	return func() {
		API("multipartbindings", func() {
		})
		basic := BasicAuthSecurity("basic")
		bearer := JWTSecurity("bearer")
		Service("uploads", func() {
			Method("send", func() {
				if location == "basic" {
					Security(basic)
				} else if location == "bearer" {
					Security(bearer)
				}
				Payload(func() {
					Attribute("files", ArrayOf(Bytes))
					if location != "none" {
						if location == "custom_query" {
							Attribute("value", String, func() {
								Pattern("^[1-9][0-9]*$")
								Meta("struct:field:type", "scalar.Token", "example.com/multipart/scalar")
							})
						} else if location == "path_string" {
							Attribute("value", String)
						} else {
							Attribute("value", Int, func() {
								Minimum(1)
							})
						}
						Required("value")
					}
					if location == "basic" {
						Username("username", String, func() {
							MinLength(3)
						})
						Password("password", String, func() {
							MinLength(3)
						})
						Required("username", "password")
					} else if location == "bearer" {
						Token("token", String)
						Required("token")
					}
					Required("files")
				})
				HTTP(func() {
					if location == "path" || location == "path_string" {
						POST("/send/{value}")
					} else {
						POST("/send")
					}
					switch location {
					case "query", "basic", "bearer", "custom_query":
						Param("value")
					case "header":
						Header("value:X-Value")
					case "cookie":
						Cookie("value")
					}
					MultipartRequest()
				})
			})
		})
	}
}
