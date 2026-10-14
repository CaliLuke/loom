package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// RequestContentLength exercises framing after raw, custom and form body encoding.
func RequestContentLength() {
	marker := Type("LengthMarker", String)
	Service("lengths", func() {
		for _, name := range []string{"optional", "required", "named"} {
			Method(name, func() {
				Payload(func() {
					if name == "named" {
						Attribute("length", marker)
					} else {
						Attribute("length", Int64)
					}
					Attribute("tags", ArrayOf(String))
					if name != "optional" {
						Required("length")
					}
				})
				HTTP(func() {
					POST("/" + name)
					Header("length:cOnTeNt-LeNgTh")
					Header("tags:X-Tags")
					SkipRequestBodyEncodeDecode()
				})
			})
		}
		for _, name := range []string{"encoded", "form", "ordinary"} {
			Method(name, func() {
				Payload(func() {
					Attribute("length", Int64)
					Attribute("value", String)
					Required("value")
				})
				HTTP(func() {
					POST("/" + name)
					Header("length:Content-Length")
					if name == "form" {
						FormRequest()
					}
				})
			})
		}
	})
}
