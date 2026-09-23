package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestJSONRPCInlineObjectBodyTransformsCompile covers inline object attributes
// nested in JSON-RPC body types. The client request and server response
// transforms declare the inline struct again as a composite literal, which
// must match the body field type, including its struct tags.
func TestJSONRPCInlineObjectBodyTransformsCompile(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		nestedElem := func() {
			Attribute("x", String)
			Attribute("y", String, func() {
				Default("d")
			})
		}
		holder := Type("Holder", func() {
			Attribute("inner", func() {
				Attribute("q", String)
				Attribute("r", String, func() {
					Default("r")
				})
			})
		})
		inline := func() {
			Attribute("holder", holder)
			Attribute("holders", ArrayOf(holder))
			Attribute("holder_index", MapOf(String, holder))
			Attribute("u", func() {
				OneOf("choice", func() {
					Attribute("s", String)
					Attribute("n", Int)
				})
				Attribute("label", String)
				Attribute("mode", String, func() {
					Default("auto")
				})
				Attribute("code", String)
				Required("code")
			})
			Attribute("list", ArrayOf(&expr.Object{}, func() {
				Attribute("x", String)
				Attribute("y", String, func() {
					Default("d")
				})
			}))
			Attribute("index", MapOf(String, &expr.Object{}, func() {
				Elem(func() {
					Attribute("x", String)
				})
			}))
			Attribute("nullable_list", ArrayOf(&expr.Object{}, func() {
				Nullable()
				Attribute("x", String)
			}))
			Attribute("nullable_index", MapOf(String, &expr.Object{}, func() {
				Elem(func() {
					Nullable()
					Attribute("x", String)
				})
			}))
			for _, nested := range []struct {
				name     string
				shape    func() any
				nullable bool
			}{
				{"nullable_grid", func() any { return ArrayOf(ArrayOf(&expr.Object{}, nestedElem)) }, true},
				{"nullable_groups", func() any { return MapOf(String, ArrayOf(&expr.Object{}, nestedElem)) }, true},
				{"sparse_grid", func() any {
					return ArrayOf(ArrayOf(&expr.Object{}, nestedElem), func() {
						Nullable()
					})
				}, false},
			} {
				if nested.nullable {
					Attribute(nested.name, nested.shape(), func() {
						Nullable()
					})
				} else {
					Attribute(nested.name, nested.shape())
				}
			}
		}
		Service("InlineTransform", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("Submit", func() {
				Payload(func() {
					ID("id", String)
					inline()
				})
				Result(func() {
					ID("id", String)
					inline()
				})
				JSONRPC(func() {})
			})
		})
	})
	dir := t.TempDir()

	renderJSONRPCModule(t, dir, "example.com/jsonrpcinlinetransform", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
}
