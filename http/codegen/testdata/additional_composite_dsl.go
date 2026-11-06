package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

type additionalCompositeCase struct {
	name string
	typ  expr.UserType
}

// AdditionalCompositeDSL reaches composite types only through additional members.
func AdditionalCompositeDSL() {
	additionalCompositeDSL(false)
}

// AdditionalCompositeJSONRPCDSL checks the same dependency graph over JSON-RPC.
func AdditionalCompositeJSONRPCDSL() {
	additionalCompositeDSL(true)
}

func additionalCompositeDSL(rpc bool) {
	API("composite", func() {
		if rpc {
			JSONRPC(func() {

			})
		}
	})
	types := append(additionalCompositeTypes(), additionalByteTypes()...)
	Service("composite", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		for _, item := range types {
			Method(item.name, func() {
				Payload(item.typ)
				Result(item.typ)
				if rpc {
					JSONRPC(func() {

					})
				} else {
					HTTP(func() {
						POST("/" + item.name)
						Response(StatusOK)
					})
				}
			})
		}
	})
}

func additionalCompositeTypes() []additionalCompositeCase {
	leaf := Type("Leaf", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	count := Type("Count", func() {
		Attribute("count", Int, func() {
			Minimum(0)
		})
		Required("count")
	})
	union := Type("Entry", OneOf(leaf, count), Untagged)
	objects := Type("Objects", func() {
		AdditionalAttributes(leaf)
		Required("x")
	})
	unions := Type("Unions", func() {
		AdditionalAttributes(union)
		Required("x")
	})
	nested := Type("Nested", func() {
		AdditionalAttributes(ArrayOf(MapOf(String, leaf)))
		Required("x")
	})
	recursive := Type("Recursive", func() {
		Attribute("name", String)
		Required("name")
		AdditionalAttributes("Recursive")
	})
	literal := Type("LiteralKeys", func() {
		Attribute("a", Int)
		AdditionalAttributes(Int)
		Required("a:b", "a,b", "", "-", "quote\"slash\\", "line\nend", "日本語", "AdditionalAttributes", "foo", "Foo")
	})
	nullable := Type("NullableKeys", func() {
		AdditionalAttributes(Int, func() {
			Nullable()
		})
		Required("value")
	})
	mapped := Type("MappedKeys", func() {
		Attribute("a:x", Int)
		AdditionalAttributes(Int)
		Required("a:x", "a:b")
	})
	occurrence := Type("OccurrenceKeys", func() {
		Attribute("record", objects, func() {
			Required("more")
		})
		Required("record")
	})
	return []additionalCompositeCase{
		{"objects", objects}, {"unions", unions}, {"nested", nested}, {"recursive", recursive},
		{"literal", literal}, {"nullable", nullable}, {"mapped", mapped}, {"occurrence", occurrence},
	}
}

func additionalByteTypes() []additionalCompositeCase {
	blob := Type("Blob", Bytes, func() {
		MinLength(2)
		MaxLength(2)
	})
	bytes := Type("ByteRecord", func() {
		Attribute("id", String)
		AdditionalAttributes(blob)
		Required("x")
		Example(Val{"id": "record", "x": []byte("hi")})
	})
	nested := Type("ByteNested", func() {
		AdditionalAttributes(ArrayOf(MapOf(String, blob)))
		Required("x")
		Example(Val{"x": []any{Val{"child": []byte("hi")}}})
	})
	nullable := Type("ByteNullable", func() {
		AdditionalAttributes(blob, Nullable)
		Required("x")
		Example(Val{"x": nil})
	})
	occurrences := Type("ByteOccurrences", func() {
		Attribute("base", bytes)
		Attribute("record", bytes, func() {
			Required("more")
		})
		Attribute("sibling", bytes, func() {
			Required("other")
		})
		Required("base", "record", "sibling")
	})
	return []additionalCompositeCase{{"bytes", bytes}, {"byteNested", nested}, {"byteNullable", nullable}, {"byteOccurrences", occurrences}}
}
