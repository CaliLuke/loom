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

func TestUntaggedByteExamplesAcrossProcesses(t *testing.T) {
	const outputEnv = "LOOM_UNTAGGED_BYTE_EXAMPLE_OUTPUT"
	if directory := os.Getenv(outputEnv); directory != "" {
		root := httpgen.RunHTTPDSL(t, testdata.MappedNamesDSL)
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
				command := exec.Command(os.Args[0], "-test.run=^TestUntaggedByteExamplesAcrossProcesses$")
				command.Env = append(os.Environ(), outputEnv+"="+directory, openAPIDeterminismHelperTarget+"="+target)
				output, err := command.CombinedOutput()
				require.NoErrorf(t, err, "isolated generation failed: %s", output)
			}
			for _, name := range []string{"openapi.json", "openapi.yaml"} {
				first, err := os.ReadFile(filepath.Join(directories[0], "gen", "http", name))
				require.NoError(t, err)
				second, err := os.ReadFile(filepath.Join(directories[1], "gen", "http", name))
				require.NoError(t, err)
				require.Equal(t, first, second)
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
				for _, contract := range []struct {
					path string
					want map[string]any
				}{
					{"/lookup", map[string]any{"dt": "payload", "b": "aGk="}},
					{"/byte-choice", map[string]any{"d": "aGk="}},
				} {
					operation := requireOperation(t, document, contract.path, "post")
					request := requireMap(t, operation["requestBody"], "request")
					responses := requireMap(t, operation["responses"], "responses")
					response := requireMap(t, responses["200"], "response")
					for _, body := range []map[string]any{request, response} {
						content := requireMap(t, body["content"], "content")
						for _, raw := range content {
							media := requireMap(t, raw, "media")
							example := requireMap(t, media["example"], "example")
							require.Equal(t, contract.want, example)
						}
					}
				}
			}
		})
	}
}
