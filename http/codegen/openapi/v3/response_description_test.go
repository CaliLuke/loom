package openapiv3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedResponseDescriptions(t *testing.T) {
	for _, version := range []struct {
		target, number string
	}{
		{"3.1", openapiv3.OpenAPICompatibilityVersion},
		{"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.OpenAPISharedErrorHeaderDSL, version.target, version.number)
			var yamlSpec map[string]any
			require.NoError(t, yaml.Unmarshal(artifacts.YAML, &yamlSpec))
			for _, spec := range []map[string]any{decodeOpenAPIJSON(t, artifacts.JSON), yamlSpec} {
				for _, method := range []string{"first", "second"} {
					operation := requireOperation(t, spec, "/"+method, "get")
					response := requireMap(t, requireMap(t, operation["responses"], "responses")["400"], "bad request")
					require.Equal(t, "#/components/responses/BadRequestError", response["$ref"])
					require.Equal(t, "bad_request: Invalid request for "+method+".\n\ninvalid_input: Invalid input for "+method+".", response["description"])
					require.Len(t, response, 2, "a reference carries only its description override")
				}
				component := requireComponentResponse(t, spec, "BadRequestError")
				require.Equal(t, "Bad Request response.", component["description"])
				responses := requireMap(t, requireMap(t, spec["components"], "components")["responses"], "response components")
				require.Len(t, responses, 4, "one per status: 204, 400, 401, 403")
			}
			if output := os.Getenv("LOOM_RESPONSE_DESCRIPTION_OUTPUT"); output != "" {
				require.NoError(t, os.WriteFile(filepath.Join(output, version.target+".json"), artifacts.JSON, 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(output, version.target+".yaml"), artifacts.YAML, 0o600))
				return
			}
			for range 2 {
				output := t.TempDir()
				command := exec.Command(os.Args[0], "-test.run=^TestRenderedResponseDescriptions$/^"+version.target+"$", "-test.count=1")
				command.Env = append(os.Environ(), "LOOM_RESPONSE_DESCRIPTION_OUTPUT="+output)
				logs, err := command.CombinedOutput()
				require.NoError(t, err, string(logs))
				for format, want := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					got, err := os.ReadFile(filepath.Join(output, version.target+"."+format))
					require.NoError(t, err)
					require.Equal(t, want, got, "independent %s generation", format)
				}
			}
		})
	}
}
