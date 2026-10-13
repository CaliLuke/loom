package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// SSEMappedAnnotationDSL exercises declared defaults and enums on mapped SSE text.
var SSEMappedAnnotationDSL = func() {
	Service("SSEMappedAnnotation", func() {
		Method("Watch", func() {
			StreamingResult(func() {
				Attribute("data", String, func() {
					Default("fallback")
					Enum("fallback", "explicit")
				})
			})
			HTTP(func() {
				GET("/mapped-annotation")
				ServerSentEvents("data")
			})
		})
	})
}
