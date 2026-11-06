package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

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
	})
	unions := Type("Unions", func() {
		AdditionalAttributes(union)
	})
	nested := Type("Nested", func() {
		AdditionalAttributes(ArrayOf(MapOf(String, leaf)))
	})
	recursive := Type("Recursive", func() {
		Attribute("name", String)
		Required("name")
		AdditionalAttributes("Recursive")
	})
	Service("composite", func() {
		if rpc {
			JSONRPC(func() {
				POST("/rpc")
			})
		}
		for _, item := range []struct {
			name string
			typ  expr.UserType
		}{
			{"objects", objects}, {"unions", unions}, {"nested", nested}, {"recursive", recursive},
		} {
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
