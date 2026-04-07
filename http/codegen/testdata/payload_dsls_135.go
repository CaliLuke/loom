package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartObjectGeneratedDSL = func() {
	Service("ServiceMultipartObjectGenerated", func() {
		Method("MethodMultipartObjectGenerated", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("content_type", String)
				Attribute("label", String, func() {
					Pattern("label-[a-z]+")
				})
				Required("file", "label")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


