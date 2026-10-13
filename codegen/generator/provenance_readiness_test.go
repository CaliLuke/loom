package generator

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
)

type provenancePluginRoot struct {
	// Value is semantic state consumed by the registered generator.
	Value string
}

// TestGenerationProvenanceIncludesPluginState checks actual plugin-owned output
// and its manifest move together when only the evaluated plugin root changes.
func TestGenerationProvenanceIncludesPluginState(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{func(_ string, roots []eval.Root) ([]*codegen.File, error) {
			for _, root := range roots {
				if plugin, ok := root.(*provenancePluginRoot); ok {
					return []*codegen.File{{Path: filepath.Join(codegen.Gendir, "plugin.txt"), Sections: []codegen.Section{codegen.NewRawSection("plugin-value", plugin.Value)}}}, nil
				}
			}
			t.Fatal("registered plugin root missing from generator inputs")
			return nil, nil
		}}, nil
	}
	digests := make([]string, 0, 2)
	outputs := make([]string, 0, 2)
	for _, value := range []string{"agent-v1", "agent-v2"} {
		codegen.RunDSL(t, func() {
			dsl.API("provenance", func() {
			})
		})
		require.NoError(t, eval.Register(&provenancePluginRoot{Value: value}))
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/provenance\n\ngo 1.27.2\n"), 0600))
		_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
		require.NoError(t, err)
		data, err := os.ReadFile(filepath.Join(dir, codegen.Gendir, "loom.json"))
		require.NoError(t, err)
		var manifest map[string]string
		require.NoError(t, json.Unmarshal(data, &manifest))
		digests = append(digests, manifest["design_digest"])
		output, err := os.ReadFile(filepath.Join(dir, codegen.Gendir, "plugin.txt"))
		require.NoError(t, err)
		outputs = append(outputs, string(output))
	}
	require.NotEqual(t, outputs[0], outputs[1], "fixture must change actual plugin output")
	require.NotEqual(t, digests[0], digests[1], "manifest must cover plugin-owned evaluated input")
}

// TestGenerationProvenanceRejectsMissingOrigin checks every command rejects
// an absent origin before allocating a generated tree.
func TestGenerationProvenanceRejectsMissingOrigin(t *testing.T) {
	codegen.RunDSL(t, func() {
		dsl.API("provenance", func() {
		})
	})
	for _, command := range []string{"gen", "example"} {
		dir := t.TempDir()
		_, err := Generate(dir, command, "", false)
		require.ErrorContains(t, err, "invalid design package")
		require.NoDirExists(t, filepath.Join(dir, codegen.Gendir))
	}
}

func (*provenancePluginRoot) EvalName() string {
	return "provenance test plugin"
}

func (*provenancePluginRoot) WalkSets(eval.SetWalker) {
}

func (*provenancePluginRoot) DependsOn() []eval.Root {
	return nil
}

func (*provenancePluginRoot) Packages() []string {
	return nil
}
