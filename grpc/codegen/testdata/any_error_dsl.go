package testdata

import . "github.com/CaliLuke/loom/dsl"

// AnyErrorDSL exercises scalar and collection Any protobuf conversion.
var AnyErrorDSL = func() {
	var Message = Type("AnyErrorMessage", func() {
		Field(1, "value", Any)
		Field(2, "values", ArrayOf(Any))
		Field(3, "mapped", MapOf(String, Any))
	})

	Service("AnyError", func() {
		Method("Convert", func() {
			Payload(Message)
			Result(Message)
			GRPC(func() {})
		})
	})
}
