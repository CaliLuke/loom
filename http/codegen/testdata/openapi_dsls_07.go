package testdata

import . "github.com/CaliLuke/loom/dsl"


var SkipResponseBodyEncodeDecodeDSL = func() {
	Service("testService", func() {
		Method("empty", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/empty")
			})
		})
		Method("empty_ok", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/empty/ok")
				Response(StatusOK)
			})
		})
		Method("binary", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/binary")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					ContentType("image/png")
				})
			})
		})
		Method("html", func() {
			Payload(Empty)
			Result(Empty)
			HTTP(func() {
				GET("/html")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					ContentType("text/html")
					OpenAPIBody(String)
				})
			})
		})
	})
}


