package openapiv3_test

import (
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const nestedUnionExampleOutput = "LOOM_NESTED_UNION_EXAMPLE_OUTPUT"

func TestNestedUnionExamplesAcrossProcesses(t *testing.T) {
	if directory := os.Getenv(nestedUnionExampleOutput); directory != "" {
		root := httpgen.RunHTTPDSL(t, testdata.TypeIdentityDSL)
		root.API.Meta = expr.MetaExpr{"openapi:version": {os.Getenv(openAPIDeterminismHelperTarget)}}
		files, err := openapiv3.Files(root)
		require.NoError(t, err)
		for _, file := range files {
			_, err := file.Render(directory)
			require.NoError(t, err)
		}
		return
	}
	for _, target := range []string{"3.1", "3.2"} {
		t.Run(target, func(t *testing.T) {
			directories := []string{t.TempDir(), t.TempDir()}
			for _, directory := range directories {
				command := exec.Command(os.Args[0], "-test.run=^TestNestedUnionExamplesAcrossProcesses$")
				command.Env = append(os.Environ(), nestedUnionExampleOutput+"="+directory, openAPIDeterminismHelperTarget+"="+target)
				output, err := command.CombinedOutput()
				require.NoErrorf(t, err, "isolated generation failed: %s", output)
			}
			for _, name := range []string{"openapi.json", "openapi.yaml"} {
				first, err := os.ReadFile(filepath.Join(directories[0], "gen", "http", name))
				require.NoError(t, err)
				second, err := os.ReadFile(filepath.Join(directories[1], "gen", "http", name))
				require.NoError(t, err)
				require.Equal(t, first, second, "examples must be byte-identical across separate processes")
				version := openapiv3.OpenAPIVersion
				if target == "3.1" {
					version = openapiv3.OpenAPICompatibilityVersion
				}
				validateOpenAPIVersion(t, first, version)
				var document map[string]any
				if filepath.Ext(name) == ".json" {
					require.NoError(t, json.Unmarshal(first, &document))
				} else {
					require.NoError(t, yaml.Unmarshal(first, &document))
				}
				record := componentSchema(t, document, "Record")
				requireNestedUnionRecord(t, record["example"])
				for _, path := range []string{"/put", "/puts"} {
					operation := requireOperation(t, document, path, "post")
					responses := requireMap(t, operation["responses"], "responses")
					response := requireMap(t, responses["200"], "success response")
					content := requireMap(t, response["content"], "content")
					for _, raw := range content {
						media := requireMap(t, raw, "media")
						example := media["example"]
						if path == "/puts" {
							items := requireSlice(t, example, "collection example")
							require.NotEmpty(t, items)
							for _, item := range items {
								requireNestedUnionRecord(t, item)
							}
						} else {
							requireNestedUnionRecord(t, example)
						}
					}
				}
			}
		})
	}
}

func requireNestedUnionRecord(t *testing.T, value any) {
	t.Helper()
	record := requireMap(t, value, "record example")
	block := requireMap(t, record["block"], "outer envelope")
	require.Contains(t, []any{"s", "t"}, block["type"])
	inner := requireMap(t, block["value"], "inner envelope")
	require.Contains(t, []any{"Leaf", "Other"}, inner["type"])
	branch := requireMap(t, inner["value"], "branch value")
	if inner["type"] == "Leaf" {
		require.IsType(t, "", branch["name"])
	} else {
		require.Contains(t, branch, "count")
	}
}
