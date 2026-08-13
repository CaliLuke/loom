package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// PayloadMultipartCompilePathParamDSL isolates generated multipart decoding
// combined with path parameter decoding.
var PayloadMultipartCompilePathParamDSL = func() {
	Service("MultipartCompilePathParam", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("project_id", String)
				Attribute("file", Bytes)
				Attribute("label", String)
				Required("project_id")
			})
			HTTP(func() {
				POST("/{project_id}")
				Param("project_id")
				MultipartRequest()
			})
		})
	})
}

// PayloadMultipartCompileRequiredFileDSL isolates generated multipart decoding
// with a required file field.
var PayloadMultipartCompileRequiredFileDSL = func() {
	Service("MultipartCompileRequiredFile", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("label", String)
				Required("file")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}

// PayloadMultipartCompileValidationDSL isolates generated multipart decoding
// with body validation.
var PayloadMultipartCompileValidationDSL = func() {
	Service("MultipartCompileValidation", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("label", String, func() {
					Pattern("label-[a-z]+")
				})
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}
