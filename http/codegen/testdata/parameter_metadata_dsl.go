package testdata

import . "github.com/CaliLuke/loom/dsl"

// ParameterMetadataDSL exercises inherited serialization flags and local overrides.
func ParameterMetadataDSL() {
	API("parameter-metadata", func() {
		Meta("openapi:version", "3.2")
	})
	text := Type("ReservedText", String, func() {
		Meta("openapi:allowReserved", "true")
	})
	alias := Type("ReservedAlias", text)
	form := Type("ReservedForm", func() {
		Attribute("labels", MapOf(String, String))
		Meta("openapi:encoding:labels:allowReserved", "true")
	})
	Service("metadata", func() {
		for _, name := range []string{"inherited", "overridden"} {
			Method(name, func() {
				Payload(func() { Attribute("value", alias) })
				Result(func() { Attribute("value", alias) })
				HTTP(func() {
					GET("/" + name)
					Param("value", func() {
						if name == "overridden" {
							Meta("openapi:allowReserved", "false")
						}
					})
					Response(StatusNoContent, func() {
						Header("value:X-Value", func() {
							if name == "overridden" {
								Meta("openapi:allowReserved", "false")
							}
						})
					})
				})
			})
			Method(name+"Form", func() {
				Payload(form)
				HTTP(func() {
					POST("/" + name + "/form")
					FormRequest()
					Body(form, func() {
						if name == "overridden" {
							Meta("openapi:encoding:labels:allowReserved", "false")
						}
					})
					Response(StatusNoContent)
				})
			})
		}
	})
}
