package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// NonASCIIServiceNamesDSL declares a service and methods whose design names
// contain non-ASCII letters, a digit followed by a lowercase letter, or a
// protocol buffer keyword. Protocol buffer identifiers are ASCII only,
// protoc-gen-go capitalizes a lowercase letter that follows a digit in Go
// names, and keywords may name rpcs unchanged.
var NonASCIIServiceNamesDSL = func() {
	Service("Café", func() {
		Method("añadir", func() {
			Payload(func() {
				Field(1, "nombre", String)
			})
			Result(func() {
				Field(1, "total", Int)
			})
			GRPC(func() {})
		})
		Method("flüss", func() {
			StreamingPayload(func() {
				Field(1, "wert", String)
			})
			StreamingResult(func() {
				Field(1, "echo", String)
			})
			GRPC(func() {})
		})
		Method("get3d", func() {
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
		Method("message", func() {
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
	})
}

// DigitServiceNamesDSL declares an ASCII service and methods whose design
// names contain a digit followed by a lowercase letter. Their protocol buffer
// names must stay as Goify produces them so that the wire paths are stable,
// while the Go names follow protoc-gen-go, which capitalizes the letter.
var DigitServiceNamesDSL = func() {
	Service("calc2go", func() {
		Method("get3d", func() {
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
		Method("v2beta_list", func() {
			Payload(String)
			Result(String)
			GRPC(func() {})
		})
		Method("sync2way", func() {
			StreamingPayload(String)
			StreamingResult(String)
			GRPC(func() {})
		})
	})
}
