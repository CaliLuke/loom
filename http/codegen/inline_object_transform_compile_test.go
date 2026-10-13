package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestGeneratedInlineObjectBodyTransformsCompile covers inline object
// attributes nested in HTTP body types. The transform that builds the body
// declares the inline struct again as a composite literal, so the literal must
// match the body field type exactly, including every struct tag.
func TestGeneratedInlineObjectBodyTransformsCompile(t *testing.T) {
	root := RunHTTPDSL(t, func() {
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
			Attribute("nested", func() {
				Attribute("inner", func() {
					Attribute("note", String)
				})
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
			Method("Submit", func() {
				Payload(func() {
					Attribute("id", String)
					inline()
					Required("id")
				})
				Result(func() {
					inline()
				})
				HTTP(func() {
					POST("/submit/{id}")
					Response(StatusOK)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/inlinetransform", root)

	for _, side := range []struct {
		name, body, transform, holder, holderTransform string
	}{
		{"client", "SubmitRequestBody", "NewSubmitRequestBody", "HolderRequestBody", "marshalInlinetransformHolderToHolderRequestBody"},
		{"server", "SubmitResponseBody", "NewSubmitResponseBody", "HolderResponseBody", "marshalInlinetransformHolderToHolderResponseBody"},
	} {
		files := parseInlineBodyPackage(t, dir, side.name)
		for _, owner := range []struct {
			body, transform string
			shapes          int
		}{
			{side.body, side.transform, 5},
			{side.holder, side.holderTransform, 1},
		} {
			t.Run(side.name+"/"+owner.body, func(t *testing.T) {
				declared, emitted := inlineBodyShapes(t, files, owner.body, owner.transform)
				require.Len(t, declared, owner.shapes, "retain the fixture's distinct anonymous shapes")
				assert.Equal(t, declared, emitted, "declaration and transform literal types must agree")
			})
		}
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
