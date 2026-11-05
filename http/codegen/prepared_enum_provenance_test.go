package codegen

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestPreparedEnumProvenanceOriginal(t *testing.T) {
	repo, err := filepath.Abs("../..")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repo)
	root := RunHTTPDSL(t, func() {
		API("objectenum", func() {
		})
		base := Type("EnumBase", func() {
			Attribute("n", Int)
			Enum(map[string]any{"n": 1}, map[string]any{"n": 2})
		})
		refined := Type("EnumRefined", func() {
			Extend(base)
			Attribute("n", Int, func() {
				Minimum(2)
			})
		})
		Service("numbers", func() {
			Method("read", func() {
				Payload(refined)
				HTTP(func() {
					POST("/numbers")
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/objectenum", root)
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012232630-0dfab0ccd366")
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}

func TestPreparedEnumProvenanceGenerated(t *testing.T) {
	repo, err := filepath.Abs("../..")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repo)
	for _, tc := range []struct {
		name, wire string
		dsl        func()
	}{
		{"implicit", "n", testdata.PreparedEnumProvenanceDSL},
		{"mapped", "wire_n", testdata.PreparedEnumMappedDSL},
		{"explicit", "n", testdata.PreparedEnumExplicitDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/objectenum", root)
			runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012232630-0dfab0ccd366")
			harness, err := os.ReadFile("testdata/prepared_enum_provenance_wire.go.txt")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(strings.ReplaceAll(string(harness), "__WIRE_NAME__", tc.wire)), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-v", "./...")
			runGoCommand(t, dir, "build", "./...")
			runGoCommand(t, dir, "vet", "./...")
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			for _, file := range files {
				path, err := file.Render(dir)
				require.NoError(t, err)
				if strings.HasSuffix(path, ".json") {
					specification, err := os.ReadFile(path)
					require.NoError(t, err)
					document, err := libopenapi.NewDocument(specification)
					require.NoError(t, err)
					_, err = document.BuildV3Model()
					require.NoError(t, err)
					assertPreparedEnumSchemas(t, specification, tc.wire)
				}
			}
		})
	}
}

func assertPreparedEnumSchemas(t *testing.T, raw []byte, wire string) {
	t.Helper()
	var specification map[string]any
	require.NoError(t, json.Unmarshal(raw, &specification))
	components := specification["components"].(map[string]any)["schemas"].(map[string]any)
	operation := specification["paths"].(map[string]any)["/numbers"].(map[string]any)["post"].(map[string]any)
	request := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"]
	response := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"]
	for _, root := range []any{request, response} {
		enumFound, minimumFound := false, false
		seen := make(map[string]bool)
		var walk func(any)
		walk = func(value any) {
			switch node := value.(type) {
			case map[string]any:
				if ref, ok := node["$ref"].(string); ok && !seen[ref] {
					seen[ref] = true
					walk(components[strings.TrimPrefix(ref, "#/components/schemas/")])
				}
				if values, ok := node["enum"].([]any); ok && len(values) == 2 {
					if first, ok := values[0].(map[string]any); ok {
						require.Equal(t, map[string]any{wire: float64(1)}, first)
						require.Equal(t, map[string]any{wire: float64(2)}, values[1])
						enumFound = true
					}
				}
				if value, ok := node["minimum"].(float64); ok && value == 2 {
					minimumFound = true
				}
				for key, child := range node {
					if key != "enum" {
						walk(child)
					}
				}
			case []any:
				for _, child := range node {
					walk(child)
				}
			}
		}
		walk(root)
		require.True(t, enumFound, "body schema must retain both declared enum constants")
		require.True(t, minimumFound, "body schema must also retain the final refinement")
	}
}
