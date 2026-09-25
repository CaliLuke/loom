package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// SkipBodyStructPkgPathDSL defines a service whose methods skip the request
// or response body encoding and use a payload and result type generated in
// the "types/common" struct:pkg:path package. The request and response data
// structures of the methods belong to the service package.
var SkipBodyStructPkgPathDSL = func() {
	item := Type("Item", func() {
		Meta("struct:pkg:path", "types/common")
		Attribute("id", String)
	})
	Service("files", func() {
		Method("upload", func() {
			Payload(item)
			Result(item)
			HTTP(func() {
				POST("/upload")
				Header("id")
				SkipRequestBodyEncodeDecode()
			})
		})
		Method("download", func() {
			Payload(item)
			Result(item)
			HTTP(func() {
				POST("/download")
				Response(StatusOK, func() {
					Header("id")
				})
				SkipResponseBodyEncodeDecode()
			})
		})
	})
}
