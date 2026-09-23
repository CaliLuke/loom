package codegen

import (
	"os"
	"path/filepath"
	"strings"
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

	clientTypes := readGeneratedTypes(t, dir, "client")
	serverTypes := readGeneratedTypes(t, dir, "server")
	for _, want := range []string{
		"`form:\"choice\" json:\"choice,omitempty\" xml:\"choice\"`",
		"Label *string `form:\"label\" json:\"label,omitempty\" xml:\"label\"`",
		"Mode string `form:\"mode\" json:\"mode\" xml:\"mode\"`",
		"Code string `form:\"code\" json:\"code\" xml:\"code\"`",
		"Note *string `form:\"note,omitempty\" json:\"note,omitempty\" xml:\"note,omitempty\"`",
		"} `form:\"inner\" json:\"inner,omitempty\" xml:\"inner\"`",
		"X *string `form:\"x\" json:\"x,omitempty\" xml:\"x\"`",
		"Y string `form:\"y\" json:\"y\" xml:\"y\"`",
	} {
		assert.GreaterOrEqual(t, strings.Count(clientTypes, want), 2,
			"client request body declaration and transform literal must agree on %s", want)
		assert.GreaterOrEqual(t, strings.Count(serverTypes, want), 2,
			"server response body declaration and transform literal must agree on %s", want)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}

// readGeneratedTypes returns the generated HTTP types of one side with each
// run of white space collapsed to one space, so that assertions do not depend
// on gofmt field alignment.
func readGeneratedTypes(t *testing.T, dir, side string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "gen", "http", "inline_transform", side, "types*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	var b strings.Builder
	for _, file := range files {
		contents, err := os.ReadFile(file)
		require.NoError(t, err)
		b.Write(contents)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
