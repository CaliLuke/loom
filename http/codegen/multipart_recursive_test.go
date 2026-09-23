package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedMultipartRecursiveTypes(t *testing.T) {
	cases := []struct {
		Name      string
		DSL       func()
		Generated bool
	}{
		{"direct-recursion", multipartRecursiveDSL(func() {
			Attribute("name", String)
			Attribute("next", "Tree")
		}), false},
		{"array-recursion", multipartRecursiveDSL(func() {
			Attribute("name", String)
			Attribute("children", ArrayOf("Tree"))
		}), false},
		{"map-recursion", multipartRecursiveDSL(func() {
			Attribute("name", String)
			Attribute("lookup", MapOf(String, "Tree"))
		}), false},
		{"repeated-non-recursive-type", multipartRepeatedTypeDSL, true},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			require.NotEmpty(t, ServerFiles("gen", services))
			require.NotEmpty(t, ClientFiles("gen", services))
			require.NotEmpty(t, ServerTypeFiles("gen", services))
			require.NotEmpty(t, ClientTypeFiles("gen", services))
			data := services.Get("uploads")
			require.Len(t, data.Endpoints, 1)
			require.Equal(t, c.Generated, data.Endpoints[0].Payload.Request.MultipartGenerated)
		})
	}
}

func multipartRecursiveDSL(tree func()) func() {
	return func() {
		var Tree = Type("Tree", tree)
		Service("uploads", func() {
			Method("upload", func() {
				Payload(func() {
					Attribute("file", Bytes)
					Attribute("tree", Tree)
					Required("file")
				})
				HTTP(func() {
					POST("/")
					MultipartRequest()
				})
			})
		})
	}
}

func multipartRepeatedTypeDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
	})
	Service("uploads", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("first", Leaf)
				Attribute("second", Leaf)
				Attribute("all", ArrayOf(Leaf))
				Required("file")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}
