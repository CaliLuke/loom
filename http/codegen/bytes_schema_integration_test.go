package codegen

import (
	_ "embed"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/expr"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/internal/schematest"
)

//go:embed testdata/bytes_schema_http_test.go.txt
var bytesSchemaHTTPHarness string

// TestBytesSchemaGeneratedHTTP checks actual server status, problem details,
// service invocation and decoded bytes. The required OpenAPI gate additionally
// compares each observation with independently validated complete emitted schemas.
func TestBytesSchemaGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.BytesSchemaHTTPDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/byteschema", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bytes_test.go"), []byte(bytesSchemaHTTPHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "1" {
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, "observations.json"))
	require.NoError(t, err)
	var observations []struct {
		Method   string `json:"method"`
		Text     string `json:"text"`
		Accepted bool   `json:"accepted"`
	}
	require.NoError(t, json.Unmarshal(data, &observations))
	require.Len(t, observations, 324)
	validator := schematest.New(t)
	controls := validator.Check(t, []schematest.Batch{{
		Schema:    map[string]any{"type": "integer", "minimum": 2},
		Instances: []any{1, 2, "2"},
	}})
	require.False(t, controls[0][0].Valid, "independent validator must reject invalid values")
	require.True(t, controls[0][1].Valid)
	require.False(t, controls[0][2].Valid, "validator must not coerce input")

	for _, version := range []string{"3.1.0", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			if root.API.Meta == nil {
				root.API.Meta = make(expr.MetaExpr)
			}
			root.API.Meta["openapi:version"] = []string{version}
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			renderGeneratedFiles(t, dir, files)
			specJSON, err := os.ReadFile(filepath.Join(dir, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			var spec map[string]any
			require.NoError(t, json.Unmarshal(specJSON, &spec))
			paths := spec["paths"].(map[string]any)
			batches := make([]schematest.Batch, 0, len(observations)*2)
			for _, observation := range observations {
				operation := paths["/"+observation.Method].(map[string]any)["post"].(map[string]any)
				content := operation["requestBody"].(map[string]any)["content"].(map[string]any)
				schema := content["application/json"].(map[string]any)["schema"].(map[string]any)
				complete := map[string]any{"allOf": []any{schema}, "components": spec["components"]}
				var value any = observation.Text
				if observation.Method == "array" {
					value = []string{observation.Text}
				} else if observation.Method == "map" {
					value = map[string]string{"key": observation.Text}
				}
				instance := map[string]any{"data": value}
				batches = append(batches, schematest.Batch{Schema: complete, Instances: []any{instance}})
				method := root.Services[0].Method(observation.Method)
				inline, err := expr.InlineJSONSchema(method.Payload)
				require.NoError(t, err)
				batches = append(batches, schematest.Batch{Schema: jsontext.Value(inline), Instances: []any{instance}})
			}
			results := validator.Check(t, batches)
			for index, observation := range observations {
				for offset, owner := range []string{"OpenAPI", "inline"} {
					result := results[index*2+offset][0]
					if observation.Accepted != result.Valid {
						t.Errorf("%s %s %q: runtime accepted=%t schema accepted=%t: %s", owner, observation.Method, observation.Text, observation.Accepted, result.Valid, result.Errors)
					}
				}
			}
		})
	}
}
