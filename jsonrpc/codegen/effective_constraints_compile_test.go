package codegen

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestEffectiveConstraintsGeneratedJSONRPCModule(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("effective", func() {
			dsl.JSONRPC(func() {})
		})
		base := dsl.Type("DefaultBase", dsl.String, func() {
			dsl.Default("base")
			dsl.MinLength(2)
		})
		inherited := dsl.Type("InheritedDefault", base)
		overridden := dsl.Type("OverriddenDefault", base, func() {
			dsl.Default("override")
		})
		baseRequired := dsl.Type("BaseRequired", func() {
			dsl.Attribute("base", dsl.String)
			dsl.Attribute("derived", dsl.String)
			dsl.Required("base")
		})
		derivedRequired := dsl.Type("DerivedRequired", baseRequired, func() {
			dsl.Required("derived")
		})
		patternBase := dsl.Type("PatternBase", dsl.String, func() { dsl.Pattern("^a") })
		patternDerived := dsl.Type("PatternDerived", patternBase, func() { dsl.Pattern("b$") })
		formatBase := dsl.Type("FormatBase", dsl.String, func() { dsl.Format(dsl.FormatIP) })
		formatDerived := dsl.Type("FormatDerived", formatBase, func() { dsl.Format(dsl.FormatIPv4) })
		dsl.Service("constraints", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("defaults", func() {
				dsl.Payload(func() {
					dsl.Attribute("inherited", inherited)
					dsl.Attribute("overridden", overridden)
					dsl.Attribute("nullable", inherited, func() {
						dsl.Nullable()
					})
				})
				dsl.JSONRPC(func() {})
			})
			dsl.Method("required", func() {
				dsl.Payload(derivedRequired)
				dsl.JSONRPC(func() {})
			})
			dsl.Method("clauses", func() {
				dsl.Payload(func() {
					dsl.Attribute("pattern", patternDerived)
					dsl.Attribute("ip", formatDerived)
					dsl.Required("pattern", "ip")
				})
				dsl.JSONRPC(func() {})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/effectiveconstraints", root)
	generated := readGeneratedGo(t, dir)
	require.Contains(t, generated, "Inherited  InheritedDefault")
	require.Contains(t, generated, "Overridden OverriddenDefault")
	require.Contains(t, generated, "loom.Nullable[InheritedDefault]")
	require.Contains(t, generated, `body.Inherited = "base"`)
	require.Contains(t, generated, `body.Overridden = "override"`)
	require.Contains(t, generated, `loom.MissingFieldError("base", "body")`)
	require.Contains(t, generated, `loom.MissingFieldError("derived", "body")`)
	require.GreaterOrEqual(t, strings.Count(generated, "loom.ValidatePattern"), 2)
	require.Contains(t, generated, `"b$"`)
	require.Contains(t, generated, `"^a"`)
	require.Contains(t, generated, `loom.FormatIPv4`)
	require.Contains(t, generated, `loom.FormatIP`)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func readGeneratedGo(t *testing.T, root string) string {
	t.Helper()
	var source strings.Builder
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source.Write(contents)
		return nil
	})
	require.NoError(t, err)
	return source.String()
}
