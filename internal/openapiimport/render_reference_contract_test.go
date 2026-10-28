package openapiimport

import (
	"encoding/json/v2"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

// These expected graphs include the existing export normalization of aliases
// and retained Extend references. Other named targets and occurrence constraints stay exact.
var recursiveSchemaGraphs = map[string]string{
	"acyclic":             `{"Node":{"properties":{"name":{"type":"string"}},"type":"object"}}`,
	"self_array":          `{"Node":{"properties":{"children":{"items":{"$ref":"#/components/schemas/Node"},"type":"array"},"name":{"type":"string"}},"type":"object"}}`,
	"mutual_map_array":    `{"Branch":{"properties":{"children":{"items":{"$ref":"#/components/schemas/Node"},"type":"array"}},"type":"object"},"Node":{"properties":{"branches":{"additionalProperties":{"$ref":"#/components/schemas/Branch"},"type":"object"}},"type":"object"}}`,
	"mixed_inheritance":   `{"Base":{"properties":{"child":{"$ref":"#/components/schemas/Node"}},"type":"object"},"Node":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"child":{"$ref":"#/components/schemas/Node"},"name":{"type":"string"}},"type":"object"}]}}`,
	"top_array_object":    `{"Branch":{"properties":{"child":{"$ref":"#/components/schemas/Node"}},"type":"object"},"Node":{"items":{"$ref":"#/components/schemas/Branch"},"type":"array"}}`,
	"alias_object":        `{"Branch":{"properties":{"child":{"$ref":"#/components/schemas/Branch"}},"type":"object"}}`,
	"nested_promoted":     `{"Node":{"properties":{"children":{"items":{"$ref":"#/components/schemas/NodeChildrenItem"},"type":"array"},"name":{"minLength":2,"type":"string"}},"required":["name"],"type":"object"},"NodeChildrenItem":{"additionalProperties":{"items":{"anyOf":[{"$ref":"#/components/schemas/Node"},{"type":"null"}]},"minItems":1,"type":"array"},"type":"object"}}`,
	"error_clones":        `{"Node":{"properties":{"child":{"$ref":"#/components/schemas/Node"},"message":{"type":"string"}},"type":"object"}}`,
	"union_forward_array": `{"Branches":{"items":{"$ref":"#/components/schemas/Leaf"},"type":"array"},"Leaf":{"properties":{"child":{"$ref":"#/components/schemas/Leaf"}},"type":"object"},"Node":{"oneOf":[{"$ref":"#/components/schemas/Branches"},{"$ref":"#/components/schemas/Leaf"}]}}`,
}

// requireRecursiveDesignGenerates verifies the reference plan through generated
// packages and parsed contracts, then compares every artifact from a fresh process.
func requireRecursiveDesignGenerates(t *testing.T, source []byte, specimen string) {
	t.Helper()
	module := requireRenderedDesignGenerates(t, source)
	vet := exec.Command("go", "vet", "-mod=mod", "./...")
	vet.Dir = module
	output, err := vet.CombinedOutput()
	require.NoError(t, err, string(output))
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		content, err := os.ReadFile(filepath.Join(module, "gen", "http", name))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(content)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
	content, err := os.ReadFile(filepath.Join(module, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract struct {
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(content, &contract))
	want, ok := recursiveSchemaGraphs[specimen]
	require.True(t, ok, "missing schema graph assertion for %s", specimen)
	var schemas map[string]any
	require.NoError(t, json.Unmarshal([]byte(want), &schemas))
	require.Equal(t, schemas, contract.Components.Schemas)
	artifacts := func() map[string][]byte {
		files := make(map[string][]byte)
		root := filepath.Join(module, "gen")
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			name, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[name] = content
			return nil
		})
		require.NoError(t, err)
		return files
	}
	before := artifacts()
	generate := exec.Command(filepath.Join(module, "loom"), "gen", "example.com/imported/design", "-o", ".")
	generate.Dir = module
	output, err = generate.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Equal(t, before, artifacts(), "independent generation must preserve every artifact byte")
}
