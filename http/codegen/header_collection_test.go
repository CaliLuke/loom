package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

//go:embed testdata/header_collection_wire.go.txt
var headerCollectionWireHarness string

// TestHeaderCollectionWire exercises logical list values across both HTTP directions.
func TestHeaderCollectionWire(t *testing.T) {
	root := RunHTTPDSL(t, headerCollectionDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_HEADER_OUTPUT"); output != "" {
		dir = filepath.Join(output, t.Name())
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderHTTPModule(t, dir, "example.com/headercollection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "header_test.go"), []byte(headerCollectionWireHarness), 0600))
	if os.Getenv("LOOM_HEADER_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-v", "-count=1", "./...")
}

// TestHeaderCollectionShapes checks defaults, required values, aliases, views and errors.
func TestHeaderCollectionShapes(t *testing.T) {
	root := RunHTTPDSL(t, headerCollectionShapesDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_HEADER_OUTPUT"); output != "" {
		dir = filepath.Join(output, t.Name())
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderHTTPModule(t, dir, "example.com/headershapes", root)
	harness, err := os.ReadFile("testdata/header_collection_shapes.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "header_test.go"), harness, 0600))
	if os.Getenv("LOOM_HEADER_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func headerCollectionDSL() {
	values := Type("Values", func() {
		Attribute("tags", ArrayOf(String))
		Attribute("numbers", ArrayOf(Int))
		Attribute("label", String)
	})
	Service("headers", func() {
		Method("echo", func() {
			Payload(values)
			Result(values)
			HTTP(func() {
				GET("/echo")
				Header("tags:X-Tags")
				Header("numbers:X-Numbers")
				Header("label:X-Label")
				Response(StatusOK, func() {
					Header("tags:X-Tags")
					Header("numbers:X-Numbers")
					Header("label:X-Label")
				})
			})
		})
	})
}

func headerCollectionShapesDSL() {
	tags := Type("Tags", ArrayOf(String))
	defaults := Type("Defaults", func() {
		Attribute("tags", tags, func() {
			Default([]string{"fallback"})
		})
		Attribute("numbers", ArrayOf(Int), func() {
			Default([]int{7})
		})
		Attribute("empty", ArrayOf(String), func() {
			Default([]string{})
		})
	})
	viewed := ResultType("application/vnd.header-view", func() {
		TypeName("HeaderView")
		Attribute("tags", ArrayOf(String))
		View("default", func() {
			Attribute("tags")
		})
		View("tiny", func() {
			Attribute("tags")
		})
	})
	Service("shapes", func() {
		Method("defaults", func() {
			Payload(defaults)
			Result(defaults)
			HTTP(func() {
				GET("/defaults")
				Header("tags:X-Tags")
				Header("numbers:X-Numbers")
				Header("empty:X-Empty")
				Response(StatusOK, func() {
					Header("tags:X-Tags")
					Header("numbers:X-Numbers")
					Header("empty:X-Empty")
				})
			})
		})
		Method("required", func() {
			Payload(func() {
				Attribute("tags", tags)
				Attribute("numbers", ArrayOf(Int), func() {
					MinLength(1)
				})
				Required("tags", "numbers")
			})
			HTTP(func() {
				GET("/required")
				Header("tags:X-Tags")
				Header("numbers:X-Numbers")
			})
		})
		Method("view", func() {
			Result(viewed)
			HTTP(func() {
				GET("/view")
				Response(StatusOK, func() {
					Header("tags:X-Tags")
				})
			})
		})
		Method("fail", func() {
			Error("bad", func() {
				Attribute("tags", tags)
			})
			HTTP(func() {
				GET("/fail")
				Response("bad", StatusBadRequest, func() {
					Header("tags:X-Tags")
				})
			})
		})
	})
}
