package testdata

import . "github.com/CaliLuke/loom/dsl"

// PathLocalNamesDSL covers path attributes that conflict with request builder
// and array conversion locals, including names that resemble allocated suffixes.
var PathLocalNamesDSL = func() {
	Service("pathlocals", func() {
		Method("raw", func() {
			Payload(func() {
				Attribute("rd", String)
				Attribute("rd2", String)
				Required("rd", "rd2")
			})
			HTTP(func() {
				POST("/raw/{rd}/{rd2}")
				SkipRequestBodyEncodeDecode()
			})
		})
		Method("stream", func() {
			Payload(func() {
				Attribute("scheme", String)
				Attribute("scheme2", String)
				Required("scheme", "scheme2")
			})
			StreamingResult(String)
			HTTP(func() {
				GET("/stream/{scheme}/{scheme2}")
			})
		})
		Method("array", func() {
			Payload(func() {
				Attribute("i", ArrayOf(Int))
				Attribute("i2", String)
				Required("i", "i2")
			})
			HTTP(func() {
				GET("/array/{i}/{i2}")
			})
		})
	})
}
