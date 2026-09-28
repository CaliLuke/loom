package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// CollectionEnumDSL exercises whole-collection enums on request and response
// bodies, including named arrays and the distinct byte-string representation.
var CollectionEnumDSL = func() {
	labels := Type("LabelList", ArrayOf(String), func() {
		Enum([]string{"val"}, []string{"a", "b"})
	})
	allBytes := make([]byte, 256)
	for index := range allBytes {
		allBytes[index] = byte(index)
	}
	bytes := Type("Blob", Bytes, func() {
		Enum("\x00\x7f", allBytes)
	})
	nested := Type("Rows", ArrayOf(ArrayOf(String)), func() {
		Enum([][]string{{"val"}})
	})
	precise := Type("PreciseValues", ArrayOf(Float32), func() {
		Enum([]float64{1.23456789})
	})
	record := Type("Record", func() {
		Attribute("rating", Float32)
		Attribute("data", Bytes)
		Required("rating", "data")
	})
	records := Type("Records", ArrayOf(record), func() {
		Enum([]struct {
			Rating float64 `json:"rating"`
			Data   string  `json:"data"`
		}{{Rating: 1.23456789, Data: "ab"}})
	})
	Service("enums", func() {
		Method("records", func() {
			NoSecurity()
			Payload(records)
			Result(records)
			HTTP(func() {
				POST("/records")
			})
		})
		Method("precise", func() {
			NoSecurity()
			Payload(precise)
			Result(precise)
			HTTP(func() {
				POST("/precise")
			})
		})
		Method("labels", func() {
			NoSecurity()
			Payload(labels)
			Result(labels)
			HTTP(func() {
				POST("/labels")
			})
		})
		Method("bytes", func() {
			NoSecurity()
			Payload(bytes)
			Result(bytes)
			HTTP(func() {
				POST("/bytes")
			})
		})
		Method("nested", func() {
			NoSecurity()
			Payload(nested)
			Result(nested)
			HTTP(func() {
				POST("/nested")
			})
		})
		Method("optional", func() {
			NoSecurity()
			Payload(func() {
				Attribute("labels", labels)
			})
			HTTP(func() {
				POST("/optional")
			})
		})
	})
	Service("enum_grpc", func() {
		Method("send", func() {
			NoSecurity()
			Payload(func() {
				Attribute("items", ArrayOf(String), func() {
					Enum([]string{"val"})
					Meta("rpc:tag", "1")
				})
				Required("items")
			})
			GRPC(func() {
				Response(CodeOK)
			})
		})
	})
}
