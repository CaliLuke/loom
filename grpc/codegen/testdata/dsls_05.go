package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var DefaultFieldsDSL = func() {
	Service("DefaultFields", func() {
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
