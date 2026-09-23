package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// MetadataVarNameCollisionDSL declares metadata, header, and trailer
// attributes whose names Goify to the same Go identifier within one endpoint,
// plus a second endpoint that reuses one of those names.
var MetadataVarNameCollisionDSL = func() {
	Service("MetadataCollision", func() {
		Method("Colliding", func() {
			Payload(func() {
				Field(1, "foo_bar", String)
				Field(2, "fooBar", String)
				Field(3, "body", String)
			})
			Result(func() {
				Field(1, "res_id", String)
				Field(2, "resId", String)
				Field(3, "body", String)
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("foo_bar")
					Attribute("fooBar:x-foo-bar")
				})
				Response(CodeOK, func() {
					Headers(func() {
						Attribute("res_id")
					})
					Trailers(func() {
						Attribute("resId:x-res-id")
					})
				})
			})
		})
		Method("Isolated", func() {
			Payload(func() {
				Field(1, "fooBar", String)
				Field(2, "body", String)
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("fooBar")
				})
			})
		})
	})
}
