package testdata

import dsl "github.com/CaliLuke/loom/dsl"

// DefaultBytesDSL exercises optional bytes defaults across protobuf conversion.
func DefaultBytesDSL() {
	state := dsl.Type("State", func() {
		dsl.Field(1, "data", dsl.Bytes, func() {
			dsl.Default([]byte("fallback"))
		})
	})
	dsl.Service("bytevalues", func() {
		dsl.Method("echo", func() {
			dsl.Payload(state)
			dsl.Result(state)
			dsl.GRPC(func() {
			})
		})
	})
}
