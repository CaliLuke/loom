package testdata

import . "github.com/CaliLuke/loom/dsl"

// AnyErrorDSL exercises unary, streaming, typed-error, and collection Any protobuf conversion.
var AnyErrorDSL = func() {
	var Message = Type("AnyErrorMessage", func() {
		Field(1, "value", Any)
		Field(2, "values", ArrayOf(Any))
		Field(3, "mapped", MapOf(String, Any))
	})
	var Failure = Type("AnyErrorFailure", func() {
		Field(1, "detail", Any)
	})

	Service("AnyError", func() {
		Method("Convert", func() {
			Payload(Message)
			Result(Message)
			Error("failure", Failure)
			GRPC(func() {})
		})
		Method("Stream", func() {
			StreamingPayload(Message)
			StreamingResult(Message)
			Error("failure", Failure)
			GRPC(func() {})
		})
	})
}
