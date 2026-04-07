package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartObjectGeneratedWithPathParamDSL = func() {
	Service("ServiceMultipartObjectGeneratedWithPathParam", func() {
		Method("MethodMultipartObjectGeneratedWithPathParam", func() {
			Payload(func() {
				Attribute("project_id", String, func() {
					Pattern("project-[a-z]+")
				})
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("content_type", String)
				Attribute("label", String, func() {
					Pattern("label-[a-z]+")
				})
				Required("project_id", "file", "label")
			})
			HTTP(func() {
				POST("/{project_id}")
				Param("project_id")
				MultipartRequest()
			})
		})
	})
}


