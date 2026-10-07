package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var DefaultFieldsDSL = func() {
	text := Type("PresenceText", String, func() {
		MaxLength(3)
	})
	blob := Type("PresenceBytes", Bytes, func() {
		MaxLength(3)
	})
	bytesAlias := Type("PresenceBlob", blob)
	scalars := Type("PresenceScalars", func() {
		Field(1, "text", text)
		Field(2, "flag", Boolean)
		Field(3, "integer", Int)
		Field(4, "integer32", Int32)
		Field(5, "integer64", Int64)
		Field(6, "unsigned", UInt)
		Field(7, "unsigned32", UInt32)
		Field(8, "unsigned64", UInt64)
		Field(9, "float32", Float32)
		Field(10, "float64", Float64)
		Field(11, "bytes", blob)
		Field(12, "required_default", String, func() {
			Default("fallback")
		})
		Field(13, "optional_default", String, func() {
			Default("fallback")
		})
		Field(14, "optional", String)
		Required("text", "flag", "integer", "integer32", "integer64", "unsigned", "unsigned32", "unsigned64", "float32", "float64", "bytes", "required_default")
	})
	Service("DefaultFields", func() {
		Method("Blob", func() {
			Payload(bytesAlias)
			Result(bytesAlias)
			GRPC(func() {})
		})
		Method("Scalars", func() {
			Payload(scalars)
			Result(scalars)
			GRPC(func() {})
		})
		Method("Stream", func() {
			StreamingPayload(scalars)
			StreamingResult(scalars)
			GRPC(func() {})
		})
		Method("Method", func() {
			Payload(func() {
				Field(1, "req", Int64)
				Field(2, "opt", Int64)
				Field(3, "def0", Int64, func() { Default(0) })
				Field(4, "def1", Int64, func() { Default(1) })
				Field(5, "def2", Int64, func() { Default(2) })
				Field(6, "reqs", String)
				Field(7, "opts", String)
				Field(8, "defs", String, func() { Default("!") })
				Field(9, "defe", String, func() { Default("") })
				Field(10, "rat", Float64)
				Field(11, "flt", Float64)
				Field(12, "flt0", Float64, func() { Default(0.0) })
				Field(13, "flt1", Float64, func() { Default(1.0) })
				Required("req", "reqs", "rat")
			})
			GRPC(func() {})
		})
	})
}
