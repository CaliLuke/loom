package testdata

import . "github.com/CaliLuke/loom/dsl"

// StreamingValidationDSL exercises root and element constraints on streaming
// messages, including named values and streams with an initial payload.
var StreamingValidationDSL = func() {
	tags := Type("Tags", ArrayOf(String, func() {
		MinLength(1)
	}), func() {
		MinLength(2)
		MaxLength(3)
	})
	alias := Type("TagAlias", tags)
	words := Type("Words", MapOf(String, String), func() {
		MinLength(2)
		MaxLength(3)
	})
	word := Type("Word", String, func() {
		MinLength(2)
		MaxLength(3)
	})
	Service("validated", func() {
		Method("upload", func() {
			StreamingPayload(tags)
			Result(Int)
			GRPC(func() {})
		})
		Method("envelope", func() {
			Payload(func() {
				Field(1, "label", String)
				Required("label")
			})
			StreamingPayload(tags)
			Result(Int)
			GRPC(func() {})
		})
		Method("direct", func() {
			StreamingPayload(ArrayOf(String), func() {
				MinLength(2)
				MaxLength(3)
			})
			Result(Int)
			GRPC(func() {})
		})
		Method("alias", func() {
			StreamingPayload(alias)
			Result(Int)
			GRPC(func() {})
		})
		Method("mapping", func() {
			StreamingPayload(words)
			Result(Int)
			GRPC(func() {})
		})
		Method("text", func() {
			StreamingPayload(word)
			Result(Int)
			GRPC(func() {})
		})
	})
}
