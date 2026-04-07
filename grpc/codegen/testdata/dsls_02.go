package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var PayloadWithValidationsDSL = func() {
	Service("PayloadWithValidation", func() {
		Method("method_a", func() {
			Payload(func() {
				Attribute("MetadataInt", Int, func() {
					Minimum(0)
					Maximum(100)
				})
				Attribute("MetadataString", String, func() {
					MinLength(5)
					MaxLength(10)
				})
			})
			GRPC(func() {
				Metadata(func() {
					Attribute("MetadataInt")
					Attribute("MetadataString")
				})
			})
		})
	})
}
