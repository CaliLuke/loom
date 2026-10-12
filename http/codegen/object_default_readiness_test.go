package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestObjectDefaultGenerated(t *testing.T) {
	for _, shape := range []string{"anonymous", "named", "array", "map", "struct source", "nullable", "field metadata", "nested", "scalar control", "collection control"} {
		t.Run(shape, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				leaf := Type("Leaf", func() {
					Attribute("x", Int)
				})
				container := Type("Container", func() {
					Attribute("child", leaf)
					Attribute("children", ArrayOf(leaf))
				})
				Service("defaults", func() {
					Method("send", func() {
						Payload(func() {
							switch shape {
							case "anonymous":
								Attribute("value", func() {
									Attribute("x", Int)
									Default(map[string]any{"x": 1})
								})
							case "named":
								Attribute("value", leaf, func() {
									Default(map[string]any{"x": 1})
								})
							case "array":
								Attribute("value", ArrayOf(leaf), func() {
									Default([]any{map[string]any{"x": 1}})
								})
							case "map":
								Attribute("value", MapOf(String, leaf), func() {
									Default(map[string]any{"k": map[string]any{"x": 1}})
								})
							case "struct source":
								Attribute("value", leaf, func() {
									Default(struct{ X int }{X: 1})
								})
							case "nullable":
								Attribute("value", leaf, func() {
									Nullable()
									Default(map[string]any{"x": 1})
								})
							case "field metadata":
								Attribute("value", func() {
									Attribute("x", Int, func() {
										Meta("struct:field:name", "Count")
										Meta("struct:tag:json", "wire_count")
									})
									Default(map[string]any{"x": 1})
								})
							case "nested":
								Attribute("value", container, func() {
									Default(map[string]any{"child": map[string]any{"x": 0}, "children": []any{map[string]any{"x": 1}}})
								})
							case "scalar control":
								Attribute("value", Int, func() {
									Default(1)
								})
							case "collection control":
								Attribute("value", ArrayOf(Int32), func() {
									Default([]int{1})
								})
							}
						})
						HTTP(func() {
							POST("/send")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/objectdefaults", root)
			runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0")
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
		})
	}
}

func TestObjectDefaultsTransport(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ObjectDefaultsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/objectdefaults", root)
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0")
	harness := strings.NewReplacer("TRANSPORT", "http", "RPC", "false", "SERVER_ARGS", "nil, nil").Replace(testdata.ObjectDefaultsHarness)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
