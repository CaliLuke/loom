package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartObjectGeneratedInvalidDSL = func() {
	Service("ServiceMultipartObjectGeneratedInvalid", func() {
		Method("MethodMultipartObjectGeneratedInvalid", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("count", Int)
				Required("file", "count")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


